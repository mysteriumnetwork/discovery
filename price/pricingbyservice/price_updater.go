package pricingbyservice

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/fln/pprotect"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/mysteriumnetwork/discovery/metrics"
	"github.com/mysteriumnetwork/payments/v3/units"
)

const PriceRedisKey = "DISCOVERY_CURRENT_PRICE_BY_SERVICE"

type Bound struct {
	Min, Max float64
}

type FiatPriceAPI interface {
	MystUSD() float64
}

type PriceUpdater struct {
	priceAPI      FiatPriceAPI
	demandIndexes CountryDemandIndexProvider
	priceLifetime time.Duration
	mystBound     Bound
	db            redis.UniversalClient

	lock        sync.Mutex
	lp          LatestPrices
	cfgProvider ConfigProvider

	stop chan struct{}
	once sync.Once
}

func NewPricer(
	cfgProvider ConfigProvider,
	priceAPI FiatPriceAPI,
	demandIndexes CountryDemandIndexProvider,
	priceLifetime time.Duration,
	sensibleMystBound Bound,
	db redis.UniversalClient,
) (*PriceUpdater, error) {
	pricer := &PriceUpdater{
		cfgProvider:   cfgProvider,
		priceAPI:      priceAPI,
		demandIndexes: demandIndexes,
		priceLifetime: priceLifetime,
		mystBound:     sensibleMystBound,
		stop:          make(chan struct{}),
		db:            db,
	}

	go pricer.schedulePriceUpdate(priceLifetime)
	if err := pricer.threadSafePriceUpdate(); err != nil {
		return nil, err
	}
	return pricer, nil
}

func (p *PriceUpdater) schedulePriceUpdate(priceLifetime time.Duration) {
	log.Info().Msg("price update be service started")
	for {
		select {
		case <-p.stop:
			log.Info().Msg("price update by service stopped")
			return
		case <-time.After(priceLifetime):
			pprotect.CallLoop(func() {
				err := p.threadSafePriceUpdate()
				if err != nil {
					log.Err(err).Msg("failed to update prices by service")
				}
			}, time.Second, func(val interface{}, stack []byte) {
				log.Warn().Msg("panic on scheduled price update by service: " + fmt.Sprint(val))
			})
		}
	}
}

func (p *PriceUpdater) threadSafePriceUpdate() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.updatePrices()
}

func (p *PriceUpdater) updatePrices() error {
	mystUSD, err := p.fetchMystPrice()
	if err != nil {
		return err
	}

	cfg, err := p.cfgProvider.Get()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	countryDemandIndexes, err := p.demandIndexes.DemandIndexes(ctx)
	if err != nil {
		return err
	}
	countryMultipliers := DemandBoostMultipliers(cfg, countryDemandIndexes)
	countryServiceMultipliers := DemandBoostServiceMultipliers(cfg, countryDemandIndexes)
	if countryMultipliers != nil && updateCountryModifiers(&cfg, countryMultipliers) {
		if err := p.cfgProvider.Update(cfg); err != nil {
			return fmt.Errorf("update country modifiers: %w", err)
		}
	}

	p.lp = p.generateNewLatestPrice(mystUSD, cfg, countryServiceMultipliers)

	marshalled, err := json.Marshal(p.lp)
	if err != nil {
		return err
	}

	err = p.db.Set(ctx, PriceRedisKey, string(marshalled), 0).Err()
	if err != nil {
		return err
	}

	p.submitMetrics()

	log.Info().Msgf("price update complete by service")
	return nil
}

func DemandBoostMultipliers(cfg Config, demandIndexes map[ISO3166CountryCode]float64) map[ISO3166CountryCode]float64 {
	if cfg.DemandBoost == nil {
		return nil
	}

	multipliers := make(map[ISO3166CountryCode]float64)
	for country, boostCfg := range cfg.DemandBoost.Countries {
		multipliers[country] = demandBoostMultiplier(boostCfg, demandIndexes[country])
	}

	return multipliers
}

func DemandBoostServiceMultipliers(cfg Config, demandIndexes map[ISO3166CountryCode]float64) map[ISO3166CountryCode]map[ServiceType]float64 {
	if cfg.DemandBoost == nil {
		return nil
	}

	multipliers := make(map[ISO3166CountryCode]map[ServiceType]float64)
	for country, boostCfg := range cfg.DemandBoost.Countries {
		multiplier := demandBoostMultiplier(boostCfg, demandIndexes[country])
		serviceTypes := boostCfg.ServiceTypes
		if len(serviceTypes) == 0 {
			serviceTypes = allServiceTypes
		}

		multipliers[country] = make(map[ServiceType]float64, len(serviceTypes))
		for _, serviceType := range serviceTypes {
			multipliers[country][serviceType] = multiplier
		}
	}

	return multipliers
}

func demandBoostMultiplier(boostCfg DemandBoostCountryCfg, currentDemandIndex float64) float64 {
	gapRatio := (boostCfg.TargetDemandIndex - currentDemandIndex) / boostCfg.TargetDemandIndex
	if gapRatio < 0 {
		gapRatio = 0
	}
	if gapRatio > 1 {
		gapRatio = 1
	}
	return 1 + (gapRatio * boostCfg.MaxBonus)
}

func updateCountryModifiers(cfg *Config, multipliers map[ISO3166CountryCode]float64) bool {
	modifiers := make(map[ISO3166CountryCode]Modifier, len(multipliers))
	for country, multiplier := range multipliers {
		modifiers[country] = Modifier{
			Residential: multiplier,
			Other:       multiplier,
		}
	}

	if len(cfg.CountryModifiers) == len(modifiers) {
		equal := true
		for country, modifier := range modifiers {
			if cfg.CountryModifiers[country] != modifier {
				equal = false
				break
			}
		}
		if equal {
			return false
		}
	}

	cfg.CountryModifiers = modifiers
	return true
}

func (p *PriceUpdater) submitMetrics() {
	p.submitPriceMetric("DEFAULTS", p.lp.Defaults.Current)

	for k, v := range p.lp.PerCountry {
		p.submitPriceMetric(k, v.Current)
	}
}

func (p *PriceUpdater) submitPriceMetric(country string, price *PriceByType) {
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "wireguard", "per_gib").Set(price.Other.Wireguard.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "wireguard", "per_hour").Set(price.Other.Wireguard.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "wireguard", "per_gib").Set(price.Residential.Wireguard.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "wireguard", "per_hour").Set(price.Residential.Wireguard.PricePerHourHumanReadable)

	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "scraping", "per_gib").Set(price.Other.Scraping.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "scraping", "per_hour").Set(price.Other.Scraping.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "scraping", "per_gib").Set(price.Residential.Scraping.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "scraping", "per_hour").Set(price.Residential.Scraping.PricePerHourHumanReadable)

	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "quic_scraping", "per_gib").Set(price.Other.QUICScraping.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "quic_scraping", "per_hour").Set(price.Other.QUICScraping.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "quic_scraping", "per_gib").Set(price.Residential.QUICScraping.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "quic_scraping", "per_hour").Set(price.Residential.QUICScraping.PricePerHourHumanReadable)

	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "data_transfer", "per_gib").Set(price.Other.DataTransfer.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "data_transfer", "per_hour").Set(price.Other.DataTransfer.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "data_transfer", "per_gib").Set(price.Residential.DataTransfer.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "data_transfer", "per_hour").Set(price.Residential.DataTransfer.PricePerHourHumanReadable)

	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "dvpn", "per_gib").Set(price.Other.DVPN.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "dvpn", "per_hour").Set(price.Other.DVPN.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "dvpn", "per_gib").Set(price.Residential.DVPN.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "dvpn", "per_hour").Set(price.Residential.DVPN.PricePerHourHumanReadable)

	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "monitoring", "per_gib").Set(price.Other.Monitoring.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "other", "monitoring", "per_hour").Set(price.Other.Monitoring.PricePerHourHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "monitoring", "per_gib").Set(price.Residential.Monitoring.PricePerGiBHumanReadable)
	metrics.CurrentPriceByCountry.WithLabelValues(country, "residential", "monitoring", "per_hour").Set(price.Residential.Monitoring.PricePerHourHumanReadable)

}

func (p *PriceUpdater) Stop() {
	p.once.Do(func() { close(p.stop) })
}

func (p *PriceUpdater) generateNewLatestPrice(mystUSD float64, cfg Config, multipliers map[ISO3166CountryCode]map[ServiceType]float64) LatestPrices {
	tm := time.Now().UTC()

	newLP := LatestPrices{
		Defaults:          p.generateNewDefaults(mystUSD, cfg),
		PerCountry:        p.generateNewPerCountryWithOptionalServiceMultipliers(mystUSD, cfg, multipliers),
		CurrentValidUntil: tm.Add(p.priceLifetime),
	}

	if !p.lp.isInitialized() {
		newLP.Defaults.Previous = newLP.Defaults.Current
		newLP.PreviousValidUntil = tm.Add(-p.priceLifetime)
	} else {
		newLP.Defaults.Previous = p.lp.Defaults.Current
		newLP.PreviousValidUntil = p.lp.CurrentValidUntil
	}
	return newLP
}

func (p *PriceUpdater) generateNewDefaults(mystUSD float64, cfg Config) *PriceHistory {
	ph := &PriceHistory{
		Current: &PriceByType{
			Residential: &PriceByServiceType{
				Wireguard: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, 1),
				},
				Scraping: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, 1),
				},
				QUICScraping: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerGiB, 1),
				},
				DataTransfer: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, 1),
				},
				DVPN: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerGiB, 1),
				},
				Monitoring: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerGiB, 1),
				},
			},
			Other: &PriceByServiceType{
				Wireguard: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, 1),
				},
				Scraping: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, 1),
				},
				QUICScraping: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerGiB, 1),
				},
				DataTransfer: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, 1),
				},
				DVPN: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DVPN.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DVPN.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DVPN.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DVPN.PricePerGiB, 1),
				},
				Monitoring: Price{
					PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerHour, 1),
					PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerHour, 1),
					PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerGiB, 1),
					PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerGiB, 1),
				},
			},
		},
	}
	if !p.lp.isInitialized() {
		ph.Previous = ph.Current
	} else {
		ph.Previous = p.lp.Defaults.Current
	}
	return ph
}

func (p *PriceUpdater) generateNewPerCountry(mystUSD float64, cfg Config) map[string]*PriceHistory {
	return p.generateNewPerCountryWithModifier(mystUSD, cfg, func(country ISO3166CountryCode, _ ServiceType) Modifier {
		modifier, ok := cfg.CountryModifiers[country]
		if !ok {
			return Modifier{Residential: 1, Other: 1}
		}
		return modifier
	})
}

func (p *PriceUpdater) generateNewPerCountryWithOptionalServiceMultipliers(mystUSD float64, cfg Config, multipliers map[ISO3166CountryCode]map[ServiceType]float64) map[string]*PriceHistory {
	if multipliers == nil {
		return p.generateNewPerCountry(mystUSD, cfg)
	}
	return p.generateNewPerCountryWithServiceMultipliers(mystUSD, cfg, multipliers)
}

func (p *PriceUpdater) generateNewPerCountryWithMultipliers(mystUSD float64, cfg Config, multipliers map[ISO3166CountryCode]float64) map[string]*PriceHistory {
	return p.generateNewPerCountryWithModifier(mystUSD, cfg, func(country ISO3166CountryCode, _ ServiceType) Modifier {
		multiplier, ok := multipliers[country]
		if !ok {
			multiplier = 1
		}
		return Modifier{Residential: multiplier, Other: multiplier}
	})
}

func (p *PriceUpdater) generateNewPerCountryWithServiceMultipliers(mystUSD float64, cfg Config, multipliers map[ISO3166CountryCode]map[ServiceType]float64) map[string]*PriceHistory {
	return p.generateNewPerCountryWithModifier(mystUSD, cfg, func(country ISO3166CountryCode, serviceType ServiceType) Modifier {
		multiplier := float64(1)
		if countryMultipliers, ok := multipliers[country]; ok {
			if serviceMultiplier, ok := countryMultipliers[serviceType]; ok {
				multiplier = serviceMultiplier
			}
		}
		return Modifier{Residential: multiplier, Other: multiplier}
	})
}

func (p *PriceUpdater) generateNewPerCountryWithModifier(mystUSD float64, cfg Config, modifierFor func(ISO3166CountryCode, ServiceType) Modifier) map[string]*PriceHistory {
	countries := make(map[string]*PriceHistory)
	for countryCode := range CountryCodeToName {
		wireguardMod := modifierFor(countryCode, ServiceTypeWireguard)
		scrapingMod := modifierFor(countryCode, ServiceTypeScraping)
		quicScrapingMod := modifierFor(countryCode, ServiceTypeQUICScraping)
		dataTransferMod := modifierFor(countryCode, ServiceTypeDataTransfer)
		dvpnMod := modifierFor(countryCode, ServiceTypeDVPN)
		monitoringMod := modifierFor(countryCode, ServiceTypeMonitoring)

		ph := &PriceHistory{
			Current: &PriceByType{
				Residential: &PriceByServiceType{
					Wireguard: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, wireguardMod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerHour, wireguardMod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, wireguardMod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Wireguard.PricePerGiB, wireguardMod.Residential),
					},
					Scraping: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, scrapingMod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerHour, scrapingMod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, scrapingMod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Scraping.PricePerGiB, scrapingMod.Residential),
					},
					QUICScraping: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerHour, quicScrapingMod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerHour, quicScrapingMod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerGiB, quicScrapingMod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.QUICScraping.PricePerGiB, quicScrapingMod.Residential),
					},
					DataTransfer: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, dataTransferMod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerHour, dataTransferMod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, dataTransferMod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DataTransfer.PricePerGiB, dataTransferMod.Residential),
					},
					DVPN: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerHour, dvpnMod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerHour, dvpnMod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerGiB, dvpnMod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.DVPN.PricePerGiB, dvpnMod.Residential),
					},
					Monitoring: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerHour, monitoringMod.Residential),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerHour, monitoringMod.Residential),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerGiB, monitoringMod.Residential),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Residential.Monitoring.PricePerGiB, monitoringMod.Residential),
					},
				},
				Other: &PriceByServiceType{
					Wireguard: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, wireguardMod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerHour, wireguardMod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, wireguardMod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Wireguard.PricePerGiB, wireguardMod.Other),
					},
					Scraping: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, scrapingMod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerHour, scrapingMod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, scrapingMod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Scraping.PricePerGiB, scrapingMod.Other),
					},
					QUICScraping: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerHour, quicScrapingMod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerHour, quicScrapingMod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerGiB, quicScrapingMod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.QUICScraping.PricePerGiB, quicScrapingMod.Other),
					},
					DataTransfer: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, dataTransferMod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerHour, dataTransferMod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, dataTransferMod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DataTransfer.PricePerGiB, dataTransferMod.Other),
					},
					DVPN: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DVPN.PricePerHour, dvpnMod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DVPN.PricePerHour, dvpnMod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.DVPN.PricePerGiB, dvpnMod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.DVPN.PricePerGiB, dvpnMod.Other),
					},
					Monitoring: Price{
						PricePerHour:              calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerHour, monitoringMod.Other),
						PricePerHourHumanReadable: calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerHour, monitoringMod.Other),
						PricePerGiB:               calculatePriceMYST(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerGiB, monitoringMod.Other),
						PricePerGiBHumanReadable:  calculatePriceMystFloat(mystUSD, cfg.BasePrices.Other.Monitoring.PricePerGiB, monitoringMod.Other),
					},
				},
			},
		}

		// if current exists in previous lp, take it, otherwise set it to current
		if p.lp.isInitialized() {
			older, ok := p.lp.PerCountry[string(countryCode)]
			if ok {
				ph.Previous = older.Current
			} else {
				ph.Previous = ph.Current
			}
		} else {
			ph.Previous = ph.Current
		}

		countries[string(countryCode)] = ph
	}
	return countries
}

// Take note that this is not 100% correct as we're rounding a bit due to accuracy issues with floats.
// This, however, is not important here as the accuracy will be more than good enough to a few zeroes after the dot.
func calculatePriceMYST(mystUSD, priceUSD, multiplier float64) *big.Int {
	return units.FloatEthToBigIntWei((priceUSD / mystUSD) * multiplier)
}

func calculatePriceMystFloat(mystUSD, priceUSD, multiplier float64) float64 {
	return units.BigIntWeiToFloatEth(calculatePriceMYST(mystUSD, priceUSD, multiplier))
}

func (p *PriceUpdater) fetchMystPrice() (float64, error) {
	mystUSD := p.priceAPI.MystUSD()
	if err := p.withinBounds(mystUSD); err != nil {
		return 0, err
	}

	return mystUSD, nil
}

// withinBounds used to filter out any possible nonsense that the external pricing services might return.
func (p *PriceUpdater) withinBounds(price float64) error {
	if price > p.mystBound.Max || price < p.mystBound.Min {
		return fmt.Errorf("myst exceeds sensible bounds: %.6f < %.6f(current price) < %.6f", p.mystBound.Min, price, p.mystBound.Max)
	}
	return nil
}

// LatestPrices holds two sets of prices. The Previous should be used in case
// a race condition between obtaining prices by Consumer and Provider
// upon agreement
type LatestPrices struct {
	Defaults           *PriceHistory            `json:"defaults"`
	PerCountry         map[string]*PriceHistory `json:"per_country"`
	CurrentValidUntil  time.Time                `json:"current_valid_until"`
	PreviousValidUntil time.Time                `json:"previous_valid_until"`
	CurrentServerTime  time.Time                `json:"current_server_time"`
}

func (lp LatestPrices) WithCurrentTime() LatestPrices {
	lp.CurrentServerTime = time.Now().UTC()
	return lp
}

func (lp *LatestPrices) isInitialized() bool {
	return lp.Defaults != nil
}

func (lp *LatestPrices) isValid() bool {
	return lp.CurrentValidUntil.UTC().After(time.Now().UTC())
}

type PriceHistory struct {
	Current  *PriceByType `json:"current"`
	Previous *PriceByType `json:"previous"`
}

type PriceByType struct {
	Residential *PriceByServiceType `json:"residential"`
	Other       *PriceByServiceType `json:"other"`
}

type PriceByServiceType struct {
	Wireguard    Price `json:"wireguard"`
	Scraping     Price `json:"scraping"`
	QUICScraping Price `json:"quic_scraping"`
	DataTransfer Price `json:"data_transfer"`
	DVPN         Price `json:"dvpn"`
	Monitoring   Price `json:"monitoring"`
}

type Price struct {
	PricePerHour              *big.Int `json:"price_per_hour" swaggertype:"integer"`
	PricePerHourHumanReadable float64  `json:"price_per_hour_human_readable" swaggertype:"number"`
	PricePerGiB               *big.Int `json:"price_per_gib" swaggertype:"integer"`
	PricePerGiBHumanReadable  float64  `json:"price_per_gib_human_readable" swaggertype:"number"`
}

CREATE TABLE IF NOT EXISTS pricing_config
(
	id              SERIAL PRIMARY KEY,
    cfg   jsonb     NOT NULL,
    created_at timestamp NOT NULL default now()
);

INSERT INTO pricing_config(
	cfg)
	VALUES ('{
  "base_prices": {
    "residential": {
      "price_per_hour_usd": 0.00036,
      "price_per_gib_usd": 0.06
    },
    "other": {
      "price_per_hour_usd": 0.00036,
      "price_per_gib_usd": 0.06
    }
  },
  "country_modifiers": {
    "US": {
      "residential": 1.5,
      "other": 1.2
    }
  }
}');
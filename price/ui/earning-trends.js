const state = {
  data: null,
  dataLoading: false,
  dataError: false,
  geo: null,
  filter: 'increasing',
  query: '',
  selectedCode: '',
  hoveredCode: '',
  scale: 1,
  offsetX: 0,
  offsetY: 0,
  drag: null,
  moved: false,
  paths: [],
  pathKey: '',
  drawPending: false
};

const trendClasses = { increasing: 'up', stable: 'stable', decreasing: 'down' };
const trendSymbols = { increasing: '↑', stable: '—', decreasing: '↓' };
const validIntensities = new Set(['low', 'medium', 'high']);
const mapColors = {
  increasing: { low: '#b7dfcc', medium: '#58b88b', high: '#11835d' },
  stable: { low: '#e5e9ec' },
  decreasing: { low: '#f4c5c3', medium: '#e88884', high: '#cf4c50' },
  unavailable: '#edf0f2',
  filtered: '#f3f5f6',
  border: '#ffffff',
  selected: '#6559e8',
  hovered: '#34465a'
};

const crimea = {
  type: 'Feature',
  properties: { ISO_A2_EH: 'UA', NAME_LONG: 'Ukraine' },
  geometry: { type: 'Polygon', coordinates: [[
    [33.628517,46.124935],[33.67028,46.218106],[33.801878,46.202706],[34.333934,46.069692],[34.464614,45.955347],[34.688637,45.974015],[34.809983,45.784996],[35.001891,45.729003],[35.480724,45.297797],[35.715994,45.340562],[35.833263,45.470893],[35.986013,45.362738],[36.284028,45.47663],[36.583263,45.427395],[36.638357,45.351264],[36.437266,45.27204],[36.440115,45.06745],[35.847992,44.992418],[35.506196,45.113511],[35.367198,44.943345],[35.256033,44.958686],[35.080821,44.793769],[34.720958,44.810981],[34.469005,44.729804],[34.331309,44.545478],[34.10255,44.425116],[33.720958,44.399644],[33.372325,44.576483],[33.571544,44.620551],[33.516287,44.785102],[33.61964,44.931464],[33.537446,45.117377],[33.396495,45.189765],[33.259613,45.156073],[32.899262,45.354234],[32.591482,45.320136],[32.481619,45.408881],[33.1692,45.78441],[33.691661,45.860745],[33.777029,45.929633],[33.619884,45.950751],[33.628517,46.124935]
  ]] }
};

const canvas = document.querySelector('#world');
const context = canvas.getContext('2d');
const hitCanvas = document.createElement('canvas');
const hitContext = hitCanvas.getContext('2d');
const tooltip = document.querySelector('#map-tooltip');
const searchInput = document.querySelector('#country-search');

function safeNumber(value, fallback = 0) {
  const number = Number(value);
  return Number.isFinite(number) ? number : fallback;
}

function normalizedCountry(country) {
  const rawCode = String(country?.country_code || '').toUpperCase();
  const countryCode = /^[A-Z]{2}$/.test(rawCode) ? rawCode : '';
  const trend = trendClasses[country?.trend] ? country.trend : 'stable';
  const intensity = validIntensities.has(country?.intensity) ? country.intensity : 'low';
  const name = String(country?.country || countryCode || 'Unknown country');
  return { country_code: countryCode, country: name, trend, intensity };
}

function allCountries() {
  return (state.data?.countries || []).map(normalizedCountry).filter(country => country.country_code);
}

function countryMap() {
  return new Map(allCountries().map(country => [country.country_code, country]));
}

function summaryItem(key) {
  const item = state.data?.summary?.[key] || {};
  return {
    count: Math.max(0, Math.round(safeNumber(item.count))),
    share: Math.max(0, Math.min(100, safeNumber(item.share_percentage)))
  };
}

function displayPercentage(value) {
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}

function setText(selector, value) {
  const element = document.querySelector(selector);
  if (element) element.textContent = value;
}

function setWidth(selector, value) {
  const element = document.querySelector(selector);
  if (element) element.style.width = value;
}

function strengthWord(country) {
  if (country.trend === 'stable') return 'Stable';
  const prefix = { low: 'Slight', medium: 'Moderate', high: 'Strong' }[country.intensity];
  return `${prefix} ${country.trend === 'increasing' ? 'increase' : 'decrease'}`;
}

function signalSentence(country) {
  if (country.trend === 'stable') return 'Earning potential is near the network baseline.';
  return `${strengthWord(country)} relative to the network baseline.`;
}

function dominantTrend(increasing, stable, decreasing) {
  const groups = [
    { trend: 'increasing', count: increasing.count },
    { trend: 'stable', count: stable.count },
    { trend: 'decreasing', count: decreasing.count }
  ];
  const highest = Math.max(...groups.map(group => group.count));
  const leaders = groups.filter(group => group.count === highest);
  return leaders.length === 1 ? leaders[0].trend : 'mixed';
}

function renderSummary() {
  if (state.dataLoading) {
    setText('#pulse-title', 'Reading the latest demand signals…');
    setText('#pulse-copy', 'Comparing current country earning potential with the network baseline.');
    ['up', 'stable', 'down'].forEach(id => {
      setText(`#${id}-count`, '—');
      setText(`#${id}-share`, 'Loading');
      setWidth(`#distribution-${id}`, '0%');
    });
    setText('#country-total', 'Loading…');
    document.querySelector('#distribution-bar')?.setAttribute('aria-label', 'Loading country distribution');
    return;
  }
  if (!state.data) {
    setText('#pulse-title', 'Demand signals are temporarily unavailable');
    setText('#pulse-copy', 'Country search and ranking will return when demand data can be loaded.');
    ['up', 'stable', 'down'].forEach(id => {
      setText(`#${id}-count`, '—');
      setText(`#${id}-share`, 'Unavailable');
      setWidth(`#distribution-${id}`, '0%');
    });
    setText('#country-total', 'Unavailable');
    document.querySelector('#distribution-bar')?.setAttribute('aria-label', 'Country distribution unavailable');
    return;
  }

  const increasing = summaryItem('increasing');
  const stable = summaryItem('stable');
  const decreasing = summaryItem('decreasing');
  const total = increasing.count + stable.count + decreasing.count;
  const firstIncreasing = allCountries().find(country => country.trend === 'increasing');
  const firstDecreasing = allCountries().find(country => country.trend === 'decreasing');
  const dominant = dominantTrend(increasing, stable, decreasing);

  if (dominant === 'increasing') {
    setText('#pulse-title', `${increasing.count} ${increasing.count === 1 ? 'country is' : 'countries are'} above the network baseline`);
    setText('#pulse-copy', firstIncreasing
      ? `${firstIncreasing.country} currently has the strongest above-baseline signal.`
      : 'Above-baseline signals form the largest group.');
  } else if (dominant === 'decreasing') {
    setText('#pulse-title', `${decreasing.count} ${decreasing.count === 1 ? 'country is' : 'countries are'} below the network baseline`);
    setText('#pulse-copy', firstIncreasing
      ? `${firstIncreasing.country} still has the strongest above-baseline signal.`
      : firstDecreasing
        ? `${firstDecreasing.country} has the strongest below-baseline signal.`
        : 'Below-baseline signals form the largest group.');
  } else if (dominant === 'stable') {
    setText('#pulse-title', 'Most countries are near the network baseline');
    setText('#pulse-copy', firstIncreasing
      ? `${firstIncreasing.country} has the strongest above-baseline signal.`
      : firstDecreasing
        ? `${firstDecreasing.country} has the strongest below-baseline signal.`
        : 'No country differs meaningfully from the network baseline.');
  } else {
    setText('#pulse-title', 'Demand signals are mixed across the network');
    setText('#pulse-copy', firstIncreasing
      ? `${firstIncreasing.country} currently has the strongest above-baseline signal.`
      : 'No single baseline-relative signal is dominant.');
  }

  const values = [['up', increasing], ['stable', stable], ['down', decreasing]];
  values.forEach(([id, item]) => {
    setText(`#${id}-count`, item.count);
    setText(`#${id}-share`, `${displayPercentage(item.share)}% of countries`);
    setWidth(`#distribution-${id}`, `${item.share}%`);
  });
  setText('#country-total', `${total} countries`);
  document.querySelector('#distribution-bar')?.setAttribute(
    'aria-label',
    `${displayPercentage(increasing.share)} percent increasing, ${displayPercentage(stable.share)} percent stable, ${displayPercentage(decreasing.share)} percent decreasing`
  );
}

function renderFilterCounts() {
  const available = Boolean(state.data);
  const increasing = summaryItem('increasing');
  const stable = summaryItem('stable');
  const decreasing = summaryItem('decreasing');
  document.querySelector('#filter-all-count').textContent = available ? increasing.count + stable.count + decreasing.count : '—';
  document.querySelector('#filter-up-count').textContent = available ? increasing.count : '—';
  document.querySelector('#filter-stable-count').textContent = available ? stable.count : '—';
  document.querySelector('#filter-down-count').textContent = available ? decreasing.count : '—';
  document.querySelectorAll('.filter-button').forEach(button => {
    button.setAttribute('aria-pressed', String(button.dataset.filter === state.filter));
    button.disabled = !available;
  });
  searchInput.disabled = !available;
}

function searchableText(value) {
  return String(value).normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLocaleLowerCase();
}

function visibleCountries() {
  const query = searchableText(state.query.trim());
  return allCountries().filter(country => {
    const trendMatches = state.filter === 'all' || country.trend === state.filter;
    const queryMatches = !query || searchableText(country.country).includes(query) || searchableText(country.country_code).includes(query);
    return trendMatches && queryMatches;
  });
}

function createStrengthMeter(country) {
  const meter = document.createElement('span');
  meter.className = `strength-meter ${trendClasses[country.trend]}`;
  meter.setAttribute('aria-hidden', 'true');
  if (country.trend === 'stable') {
    meter.append(document.createElement('i'));
    return meter;
  }
  const activeCount = { low: 1, medium: 2, high: 3 }[country.intensity];
  for (let index = 0; index < 3; index += 1) {
    const bar = document.createElement('i');
    if (index < activeCount) bar.className = 'active';
    meter.append(bar);
  }
  return meter;
}

function createCountryRow(country) {
  const row = document.createElement('button');
  row.type = 'button';
  row.className = 'signal-row';
  row.dataset.countryCode = country.country_code;
  row.setAttribute('aria-current', String(country.country_code === state.selectedCode));
  row.setAttribute('aria-label', `${country.country}, ${strengthWord(country)}`);

  const code = document.createElement('span');
  code.className = 'country-code';
  code.textContent = country.country_code;
  const copy = document.createElement('span');
  copy.className = 'country-copy';
  const name = document.createElement('strong');
  name.textContent = country.country;
  copy.append(name);
  const indicator = document.createElement('span');
  indicator.className = 'signal-indicator';
  const label = document.createElement('span');
  label.className = `signal-label ${trendClasses[country.trend]}`;
  label.textContent = `${trendSymbols[country.trend]} ${strengthWord(country)}`;
  indicator.append(label);
  if (country.trend !== 'stable') indicator.append(createStrengthMeter(country));
  row.append(code, copy, indicator);

  row.addEventListener('mouseenter', () => setHoveredCountry(country.country_code));
  row.addEventListener('mouseleave', () => setHoveredCountry(''));
  row.addEventListener('focus', () => setHoveredCountry(country.country_code));
  row.addEventListener('blur', () => setHoveredCountry(''));
  row.addEventListener('click', () => selectCountry(country.country_code, { focusMap: true }));
  return row;
}

function renderCountryList() {
  const list = document.querySelector('#country-list');
  list.replaceChildren();
  list.setAttribute('aria-busy', String(state.dataLoading));
  if (state.dataLoading) {
    const loading = document.createElement('div');
    loading.className = 'list-state';
    loading.textContent = 'Loading country signals…';
    list.append(loading);
    document.querySelector('#result-count').textContent = 'Loading…';
    return;
  }
  if (!state.data) {
    const error = document.createElement('div');
    error.className = 'list-state';
    const title = document.createElement('strong');
    title.textContent = 'Country signals could not be loaded';
    error.append(title, 'The map can still be explored once demand data returns.', createButton('Retry demand data', loadTrends, 'retry'));
    list.append(error);
    document.querySelector('#result-count').textContent = 'Unavailable';
    return;
  }

  const countries = visibleCountries();
  document.querySelector('#result-count').textContent = state.query
    ? `${countries.length} ${countries.length === 1 ? 'match' : 'matches'}`
    : `${countries.length} ${countries.length === 1 ? 'country' : 'countries'}`;
  if (!countries.length) {
    const empty = document.createElement('div');
    empty.className = 'list-state';
    const title = document.createElement('strong');
    title.textContent = 'No countries match this view';
    empty.append(title, 'Try another name, code, or trend.', createButton('Show all countries', resetFilters, 'reset-filter'));
    list.append(empty);
    return;
  }
  const fragment = document.createDocumentFragment();
  countries.forEach(country => fragment.append(createCountryRow(country)));
  list.append(fragment);
}

function renderData() {
  renderSummary();
  renderFilterCounts();
  renderCountryList();
  updateMapScope();
  state.pathKey = '';
  scheduleDraw();
}

function createButton(label, handler, className) {
  const button = document.createElement('button');
  button.type = 'button';
  button.className = className;
  button.textContent = label;
  button.addEventListener('click', handler);
  return button;
}

function setFilter(filter) {
  if (!['all', 'increasing', 'stable', 'decreasing'].includes(filter)) return;
  state.filter = filter;
  if (state.selectedCode && !visibleCountries().some(country => country.country_code === state.selectedCode)) clearSelection();
  renderFilterCounts();
  renderCountryList();
  updateMapScope();
  scheduleDraw();
}

function resetFilters() {
  state.filter = 'all';
  state.query = '';
  searchInput.value = '';
  document.querySelector('#clear-search').hidden = true;
  renderFilterCounts();
  renderCountryList();
  updateMapScope();
  scheduleDraw();
  searchInput.focus();
}

function updateMapScope() {
  const scope = document.querySelector('#map-scope');
  if (state.dataLoading) {
    scope.textContent = 'Loading demand data…';
  } else if (state.dataError) {
    scope.textContent = 'Demand data unavailable';
  } else if (state.query) {
    const count = visibleCountries().length;
    scope.textContent = `${count} search ${count === 1 ? 'match' : 'matches'}`;
  } else {
    scope.textContent = state.filter === 'all' ? 'Showing all countries' : `Showing ${state.filter} countries`;
  }
}

function setHoveredCountry(code) {
  if (state.hoveredCode === code) return;
  state.hoveredCode = code;
  scheduleDraw();
}

function selectCountry(code, options = {}) {
  const country = countryMap().get(code);
  if (!country) return;
  state.selectedCode = code;
  renderInspector(country);
  document.querySelector('#country-panel-foot').textContent = `${country.country} selected. Its signal is highlighted on the map.`;
  updateRowSelection();
  scheduleDraw();
  if (options.focusMap) focusCountryOnMap(code);
  if (options.revealRow) revealCountryRow(code);
}

function clearSelection() {
  state.selectedCode = '';
  document.querySelector('#country-inspector').hidden = true;
  document.querySelector('#country-panel-foot').textContent = 'Ranking is qualitative and never exposes price or numeric change.';
  updateRowSelection();
  scheduleDraw();
}

function updateRowSelection() {
  document.querySelectorAll('.signal-row').forEach(row => {
    row.setAttribute('aria-current', String(row.dataset.countryCode === state.selectedCode));
  });
}

function revealCountryRow(code) {
  const list = document.querySelector('#country-list');
  const row = Array.from(document.querySelectorAll('.signal-row')).find(candidate => candidate.dataset.countryCode === code);
  if (!row) return;
  const listRect = list.getBoundingClientRect();
  const rowRect = row.getBoundingClientRect();
  if (rowRect.top < listRect.top) list.scrollTop -= listRect.top - rowRect.top;
  else if (rowRect.bottom > listRect.bottom) list.scrollTop += rowRect.bottom - listRect.bottom;
}

function renderInspector(country) {
  const inspector = document.querySelector('#country-inspector');
  document.querySelector('#inspector-code').textContent = `${country.country_code} · Selected country`;
  document.querySelector('#inspector-country').textContent = country.country;
  const signal = document.querySelector('#inspector-signal');
  signal.className = `inspector-signal ${trendClasses[country.trend]}`;
  signal.textContent = `${trendSymbols[country.trend]} ${strengthWord(country)}`;
  document.querySelector('#inspector-copy').textContent = signalSentence(country);
  inspector.hidden = false;
}

function isoCode(feature) {
  return String(feature?.properties?.ISO_A2_EH || feature?.properties?.ISO_A2 || '').toUpperCase();
}

function project(point, width, height) {
  const topLatitude = 84;
  const bottomLatitude = -60;
  const mapHeight = Math.max(1, height - 24);
  const top = (height - mapHeight) / 2;
  const latitude = Math.max(bottomLatitude, Math.min(topLatitude, point[1]));
  return [(point[0] + 180) / 360 * width, top + (topLatitude - latitude) / (topLatitude - bottomLatitude) * mapHeight];
}

function polygonPath(target, coordinates, width, height) {
  coordinates.forEach(ring => {
    ring.forEach((point, index) => {
      const [x, y] = project(point, width, height);
      if (index) target.lineTo(x, y); else target.moveTo(x, y);
    });
    target.closePath();
  });
}

function featurePath(target, feature, width, height) {
  const geometry = feature.geometry;
  if (!geometry) return;
  if (geometry.type === 'Polygon') polygonPath(target, geometry.coordinates, width, height);
  if (geometry.type === 'MultiPolygon') geometry.coordinates.forEach(polygon => polygonPath(target, polygon, width, height));
}

function walkCoordinates(value, visit) {
  if (!Array.isArray(value)) return;
  if (typeof value[0] === 'number' && typeof value[1] === 'number') {
    visit(value);
    return;
  }
  value.forEach(child => walkCoordinates(child, visit));
}

function featureCenter(feature, width, height) {
  const bounds = { left: Infinity, right: -Infinity, top: Infinity, bottom: -Infinity };
  walkCoordinates(feature.geometry?.coordinates, point => {
    const [x, y] = project(point, width, height);
    bounds.left = Math.min(bounds.left, x);
    bounds.right = Math.max(bounds.right, x);
    bounds.top = Math.min(bounds.top, y);
    bounds.bottom = Math.max(bounds.bottom, y);
  });
  return Number.isFinite(bounds.left)
    ? { x: (bounds.left + bounds.right) / 2, y: (bounds.top + bounds.bottom) / 2 }
    : { x: width / 2, y: height / 2 };
}

function ensurePaths() {
  if (!state.geo) return null;
  const box = canvas.getBoundingClientRect();
  const ratio = window.devicePixelRatio || 1;
  const width = Math.max(1, Math.round(box.width));
  const height = Math.max(1, Math.round(box.height));
  const key = `${width}:${height}`;
  if (canvas.width !== Math.round(width * ratio) || canvas.height !== Math.round(height * ratio)) {
    canvas.width = Math.round(width * ratio);
    canvas.height = Math.round(height * ratio);
  }
  if (state.pathKey === key && state.paths.length) return { width, height, ratio };

  state.paths = state.geo.features
    .filter(feature => /^[A-Z]{2}$/.test(isoCode(feature)) && isoCode(feature) !== 'AQ')
    .map(feature => {
      const path = new Path2D();
      featurePath(path, feature, width, height);
      return {
        path,
        code: isoCode(feature),
        name: String(feature.properties.NAME_LONG || feature.properties.NAME || isoCode(feature)),
        center: featureCenter(feature, width, height)
      };
    });
  const crimeaPath = new Path2D();
  featurePath(crimeaPath, crimea, width, height);
  state.paths.push({ path: crimeaPath, code: 'UA', name: 'Ukraine', center: featureCenter(crimea, width, height) });
  state.pathKey = key;
  return { width, height, ratio };
}

function filteredCountryCodes() {
  return new Set(visibleCountries().map(country => country.country_code));
}

function isMapCountryVisible(code) {
  return Boolean(state.data) && filteredCountryCodes().has(code);
}

function fillColor(country) {
  return mapColors[country.trend]?.[country.intensity] || mapColors[country.trend]?.low || mapColors.stable.low;
}

function clampOffset(width, height) {
  const maxX = Math.max(0, width * (state.scale - 1) / 2 + 70);
  const maxY = Math.max(0, height * (state.scale - 1) / 2 + 70);
  state.offsetX = Math.max(-maxX, Math.min(maxX, state.offsetX));
  state.offsetY = Math.max(-maxY, Math.min(maxY, state.offsetY));
}

function draw() {
  state.drawPending = false;
  const dimensions = ensurePaths();
  if (!dimensions) return;
  const { width, height, ratio } = dimensions;
  clampOffset(width, height);
  context.setTransform(ratio, 0, 0, ratio, 0, 0);
  context.clearRect(0, 0, width, height);
  context.save();
  context.translate(state.offsetX + width / 2, state.offsetY + height / 2);
  context.scale(state.scale, state.scale);
  context.translate(-width / 2, -height / 2);

  const countries = countryMap();
  const visibleCodes = filteredCountryCodes();
  state.paths.forEach(item => {
    const country = countries.get(item.code) || { trend: 'stable', intensity: 'low', country: item.name, country_code: item.code };
    if (state.dataError || !state.data) context.fillStyle = mapColors.unavailable;
    else context.fillStyle = visibleCodes.has(item.code) ? fillColor(country) : mapColors.filtered;
    context.fill(item.path, 'evenodd');
    context.strokeStyle = mapColors.border;
    context.lineWidth = .75 / state.scale;
    context.stroke(item.path);
  });
  state.paths.forEach(item => {
    const selected = item.code === state.selectedCode;
    const hovered = item.code === state.hoveredCode;
    if (!selected && !hovered) return;
    context.strokeStyle = selected ? mapColors.selected : mapColors.hovered;
    context.lineWidth = (selected ? 2.3 : 1.45) / state.scale;
    context.stroke(item.path);
  });
  context.restore();
}

function scheduleDraw() {
  if (state.drawPending) return;
  state.drawPending = true;
  requestAnimationFrame(draw);
}

function mapPoint(event) {
  const rect = canvas.getBoundingClientRect();
  return {
    x: (event.clientX - rect.left - state.offsetX - rect.width / 2) / state.scale + rect.width / 2,
    y: (event.clientY - rect.top - state.offsetY - rect.height / 2) / state.scale + rect.height / 2
  };
}

function hitCountry(event) {
  const point = mapPoint(event);
  for (let index = state.paths.length - 1; index >= 0; index -= 1) {
    if (hitContext.isPointInPath(state.paths[index].path, point.x, point.y, 'evenodd')) return state.paths[index];
  }
  return null;
}

function showTooltip(event, item) {
  if (!item || state.dataError || !isMapCountryVisible(item.code)) {
    tooltip.style.display = 'none';
    return;
  }
  const country = countryMap().get(item.code);
  if (!country) {
    tooltip.style.display = 'none';
    return;
  }
  const title = document.createElement('strong');
  title.textContent = country.country;
  const signal = document.createElement('span');
  signal.textContent = `${trendSymbols[country.trend]} ${signalSentence(country)}`;
  tooltip.replaceChildren(title, signal);
  const rect = canvas.getBoundingClientRect();
  const localX = event.clientX - rect.left;
  const localY = event.clientY - rect.top;
  tooltip.style.display = 'block';
  tooltip.style.left = `${Math.max(8, Math.min(localX + 14, canvas.clientWidth - 240))}px`;
  tooltip.style.top = `${Math.max(8, localY - 58)}px`;
}

function focusCountryOnMap(code) {
  const dimensions = ensurePaths();
  if (!dimensions) return;
  const item = state.paths.find(path => path.code === code && path.code !== 'UA') || state.paths.find(path => path.code === code);
  if (!item) return;
  state.scale = Math.max(1.75, state.scale);
  state.offsetX = -(item.center.x - dimensions.width / 2) * state.scale;
  state.offsetY = -(item.center.y - dimensions.height / 2) * state.scale;
  scheduleDraw();
}

function zoom(factor) {
  state.scale = Math.max(1, Math.min(4, state.scale * factor));
  if (state.scale === 1) {
    state.offsetX = 0;
    state.offsetY = 0;
  }
  scheduleDraw();
}

function resetMap() {
  state.scale = 1;
  state.offsetX = 0;
  state.offsetY = 0;
  scheduleDraw();
}

function inspectHighestVisibleCountry() {
  const drawable = new Set(state.paths.map(path => path.code));
  const country = visibleCountries().find(candidate => drawable.has(candidate.country_code));
  if (country) selectCountry(country.country_code, { focusMap: true, revealRow: true });
}

document.querySelectorAll('.filter-button').forEach(button => {
  button.addEventListener('click', () => setFilter(button.dataset.filter));
});

searchInput.addEventListener('input', event => {
  state.query = event.target.value;
  document.querySelector('#clear-search').hidden = !state.query;
  if (state.selectedCode && !visibleCountries().some(country => country.country_code === state.selectedCode)) clearSelection();
  renderFilterCounts();
  renderCountryList();
  updateMapScope();
  scheduleDraw();
});

document.querySelector('#clear-search').addEventListener('click', () => {
  state.query = '';
  searchInput.value = '';
  document.querySelector('#clear-search').hidden = true;
  renderCountryList();
  updateMapScope();
  scheduleDraw();
  searchInput.focus();
});
document.querySelector('#inspector-close').addEventListener('click', clearSelection);
document.querySelector('#zoom-in').addEventListener('click', () => zoom(1.3));
document.querySelector('#zoom-out').addEventListener('click', () => zoom(1 / 1.3));
document.querySelector('#reset-map').addEventListener('click', resetMap);

canvas.addEventListener('pointerdown', event => {
  canvas.setPointerCapture(event.pointerId);
  state.drag = { x: event.clientX, y: event.clientY };
  state.moved = false;
  tooltip.style.display = 'none';
});
canvas.addEventListener('pointermove', event => {
  if (state.drag) {
    const deltaX = event.clientX - state.drag.x;
    const deltaY = event.clientY - state.drag.y;
    state.offsetX += deltaX;
    state.offsetY += deltaY;
    state.moved = state.moved || Math.abs(deltaX) + Math.abs(deltaY) > 2;
    state.drag = { x: event.clientX, y: event.clientY };
    scheduleDraw();
    return;
  }
  if (event.pointerType === 'mouse' || (event.pointerType === 'pen' && event.buttons === 0)) {
    const candidate = hitCountry(event);
    const item = candidate && isMapCountryVisible(candidate.code) ? candidate : null;
    setHoveredCountry(item?.code || '');
    if (event.pointerType === 'mouse') showTooltip(event, item);
    else tooltip.style.display = 'none';
  }
});
canvas.addEventListener('pointerup', event => {
  if (!state.moved) {
    const candidate = hitCountry(event);
    const item = candidate && isMapCountryVisible(candidate.code) ? candidate : null;
    if (item) selectCountry(item.code, { revealRow: true });
    if (event.pointerType === 'mouse') showTooltip(event, item);
    else tooltip.style.display = 'none';
  }
  state.drag = null;
  if (canvas.hasPointerCapture(event.pointerId)) canvas.releasePointerCapture(event.pointerId);
});
canvas.addEventListener('pointercancel', () => {
  state.drag = null;
  state.moved = false;
  tooltip.style.display = 'none';
});
canvas.addEventListener('lostpointercapture', () => { state.drag = null; });
canvas.addEventListener('pointerleave', () => {
  if (!state.drag) {
    tooltip.style.display = 'none';
    setHoveredCountry('');
  }
});
canvas.addEventListener('wheel', event => {
  event.preventDefault();
  zoom(event.deltaY < 0 ? 1.12 : 1 / 1.12);
}, { passive: false });
canvas.addEventListener('keydown', event => {
  const step = 28;
  if (event.key === 'ArrowLeft') state.offsetX += step;
  else if (event.key === 'ArrowRight') state.offsetX -= step;
  else if (event.key === 'ArrowUp') state.offsetY += step;
  else if (event.key === 'ArrowDown') state.offsetY -= step;
  else if (event.key === '+' || event.key === '=') zoom(1.25);
  else if (event.key === '-') zoom(.8);
  else if (event.key === 'Home' || event.key === '0') resetMap();
  else if (event.key === 'Enter' || event.key === ' ') inspectHighestVisibleCountry();
  else if (event.key === 'Escape') clearSelection();
  else return;
  event.preventDefault();
  scheduleDraw();
});

document.addEventListener('keydown', event => {
  if (event.key !== 'Escape' || document.activeElement === canvas) return;
  if (state.query) {
    state.query = '';
    searchInput.value = '';
    document.querySelector('#clear-search').hidden = true;
    renderCountryList();
    updateMapScope();
  }
  clearSelection();
});

async function loadTrends() {
  if (state.dataLoading) return;
  state.dataLoading = true;
  state.dataError = false;
  state.data = null;
  clearSelection();
  renderData();
  try {
    const response = await fetch('/api/v4/earning-trends?scope=countries', { headers: { Accept: 'application/json' } });
    if (!response.ok) throw new Error('Demand request failed');
    const data = await response.json();
    if (!data || !data.summary || !Array.isArray(data.countries)) throw new Error('Invalid demand response');
    state.data = data;
  } catch (error) {
    state.data = null;
    state.dataError = true;
  }
  state.dataLoading = false;
  renderData();
}

async function loadMap() {
  const status = document.querySelector('#map-status');
  status.hidden = false;
  status.replaceChildren(document.createTextNode('Loading map…'));
  try {
    const response = await fetch('/api/v4/earning-trends/world.geojson');
    if (!response.ok) throw new Error('Map request failed');
    const geo = await response.json();
    if (!geo || !Array.isArray(geo.features)) throw new Error('Invalid map response');
    state.geo = geo;
    state.pathKey = '';
    status.hidden = true;
    scheduleDraw();
  } catch (error) {
    state.geo = null;
    status.replaceChildren(document.createTextNode('The map could not be loaded.'), createButton('Retry map', loadMap, 'retry'));
  }
}

if ('ResizeObserver' in window) {
  new ResizeObserver(() => {
    state.pathKey = '';
    scheduleDraw();
  }).observe(canvas);
} else {
  window.addEventListener('resize', () => {
    state.pathKey = '';
    scheduleDraw();
  });
}

loadTrends();
loadMap();

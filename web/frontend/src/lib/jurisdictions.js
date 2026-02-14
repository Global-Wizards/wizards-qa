export const JURISDICTIONS = [
  // North America
  { id: 'us-nj', name: 'New Jersey', country: 'United States', region: 'North America', regulator: 'NJDGE' },
  { id: 'us-nv', name: 'Nevada', country: 'United States', region: 'North America', regulator: 'NGC' },
  { id: 'us-pa', name: 'Pennsylvania', country: 'United States', region: 'North America', regulator: 'PGCB' },
  { id: 'us-mi', name: 'Michigan', country: 'United States', region: 'North America', regulator: 'MGCB' },
  { id: 'us-wv', name: 'West Virginia', country: 'United States', region: 'North America', regulator: 'WVLRC' },
  { id: 'us-ct', name: 'Connecticut', country: 'United States', region: 'North America', regulator: 'DOSR' },
  { id: 'us-de', name: 'Delaware', country: 'United States', region: 'North America', regulator: 'DLG' },
  { id: 'us-ri', name: 'Rhode Island', country: 'United States', region: 'North America', regulator: 'RIDBL' },
  { id: 'ca-on', name: 'Ontario', country: 'Canada', region: 'North America', regulator: 'AGCO' },
  { id: 'ca-bc', name: 'British Columbia', country: 'Canada', region: 'North America', regulator: 'GPEB' },
  { id: 'ca-ab', name: 'Alberta', country: 'Canada', region: 'North America', regulator: 'AGLC' },
  { id: 'ca-qc', name: 'Quebec', country: 'Canada', region: 'North America', regulator: 'RAJ' },
  { id: 'mx', name: 'Mexico', country: 'Mexico', region: 'North America', regulator: 'SEGOB' },

  // Europe
  { id: 'gb', name: 'United Kingdom', country: 'United Kingdom', region: 'Europe', regulator: 'UKGC' },
  { id: 'mt', name: 'Malta', country: 'Malta', region: 'Europe', regulator: 'MGA' },
  { id: 'gi', name: 'Gibraltar', country: 'Gibraltar', region: 'Europe', regulator: 'GRA' },
  { id: 'im', name: 'Isle of Man', country: 'Isle of Man', region: 'Europe', regulator: 'GSC' },
  { id: 'gg', name: 'Alderney', country: 'Alderney', region: 'Europe', regulator: 'AGCC' },
  { id: 'se', name: 'Sweden', country: 'Sweden', region: 'Europe', regulator: 'SGA' },
  { id: 'dk', name: 'Denmark', country: 'Denmark', region: 'Europe', regulator: 'DGA' },
  { id: 'es', name: 'Spain', country: 'Spain', region: 'Europe', regulator: 'DGOJ' },
  { id: 'it', name: 'Italy', country: 'Italy', region: 'Europe', regulator: 'ADM' },
  { id: 'fr', name: 'France', country: 'France', region: 'Europe', regulator: 'ANJ' },
  { id: 'pt', name: 'Portugal', country: 'Portugal', region: 'Europe', regulator: 'SRIJ' },
  { id: 'nl', name: 'Netherlands', country: 'Netherlands', region: 'Europe', regulator: 'KSA' },
  { id: 'be', name: 'Belgium', country: 'Belgium', region: 'Europe', regulator: 'GCB' },
  { id: 'de', name: 'Germany', country: 'Germany', region: 'Europe', regulator: 'GGL' },
  { id: 'ro', name: 'Romania', country: 'Romania', region: 'Europe', regulator: 'ONJN' },
  { id: 'bg', name: 'Bulgaria', country: 'Bulgaria', region: 'Europe', regulator: 'SCG' },
  { id: 'cz', name: 'Czech Republic', country: 'Czech Republic', region: 'Europe', regulator: 'MoF' },
  { id: 'ee', name: 'Estonia', country: 'Estonia', region: 'Europe', regulator: 'EMTA' },
  { id: 'lv', name: 'Latvia', country: 'Latvia', region: 'Europe', regulator: 'IAUI' },
  { id: 'lt', name: 'Lithuania', country: 'Lithuania', region: 'Europe', regulator: 'GCC' },
  { id: 'gr', name: 'Greece', country: 'Greece', region: 'Europe', regulator: 'HGC' },
  { id: 'at', name: 'Austria', country: 'Austria', region: 'Europe', regulator: 'BMF' },
  { id: 'ch', name: 'Switzerland', country: 'Switzerland', region: 'Europe', regulator: 'ESBK' },
  { id: 'ie', name: 'Ireland', country: 'Ireland', region: 'Europe', regulator: 'GRAI' },
  { id: 'fi', name: 'Finland', country: 'Finland', region: 'Europe', regulator: 'Poliisihallitus' },
  { id: 'no', name: 'Norway', country: 'Norway', region: 'Europe', regulator: 'Lotteritilsynet' },
  { id: 'hr', name: 'Croatia', country: 'Croatia', region: 'Europe', regulator: 'MoF' },
  { id: 'rs', name: 'Serbia', country: 'Serbia', region: 'Europe', regulator: 'GIA' },

  // Latin America & Caribbean
  { id: 'br', name: 'Brazil', country: 'Brazil', region: 'Latin America & Caribbean', regulator: 'SPA/MF' },
  { id: 'co', name: 'Colombia', country: 'Colombia', region: 'Latin America & Caribbean', regulator: 'Coljuegos' },
  { id: 'ar-ba', name: 'Buenos Aires', country: 'Argentina', region: 'Latin America & Caribbean', regulator: 'LOTBA' },
  { id: 'ar-co', name: 'Cordoba', country: 'Argentina', region: 'Latin America & Caribbean', regulator: 'LOTCOR' },
  { id: 'pe', name: 'Peru', country: 'Peru', region: 'Latin America & Caribbean', regulator: 'MINCETUR' },
  { id: 'cl', name: 'Chile', country: 'Chile', region: 'Latin America & Caribbean', regulator: 'SCJ' },
  { id: 'pa', name: 'Panama', country: 'Panama', region: 'Latin America & Caribbean', regulator: 'JCJ' },
  { id: 'cr', name: 'Costa Rica', country: 'Costa Rica', region: 'Latin America & Caribbean', regulator: 'N/A' },
  { id: 'gt', name: 'Guatemala', country: 'Guatemala', region: 'Latin America & Caribbean', regulator: 'N/A' },
  { id: 'hn', name: 'Honduras', country: 'Honduras', region: 'Latin America & Caribbean', regulator: 'N/A' },
  { id: 'sv', name: 'El Salvador', country: 'El Salvador', region: 'Latin America & Caribbean', regulator: 'N/A' },
  { id: 'cw', name: 'Curacao', country: 'Curacao', region: 'Latin America & Caribbean', regulator: 'GCB' },
  { id: 'ag', name: 'Antigua & Barbuda', country: 'Antigua & Barbuda', region: 'Latin America & Caribbean', regulator: 'FSRC' },
  { id: 'jm', name: 'Jamaica', country: 'Jamaica', region: 'Latin America & Caribbean', regulator: 'BGLC' },

  // Asia-Pacific
  { id: 'ph', name: 'Philippines', country: 'Philippines', region: 'Asia-Pacific', regulator: 'PAGCOR' },
  { id: 'mo', name: 'Macau', country: 'Macau', region: 'Asia-Pacific', regulator: 'DICJ' },
  { id: 'jp', name: 'Japan', country: 'Japan', region: 'Asia-Pacific', regulator: 'JCC' },
  { id: 'kr', name: 'South Korea', country: 'South Korea', region: 'Asia-Pacific', regulator: 'NRC' },
  { id: 'au-nsw', name: 'New South Wales', country: 'Australia', region: 'Asia-Pacific', regulator: 'LGNSW' },
  { id: 'au-vic', name: 'Victoria', country: 'Australia', region: 'Asia-Pacific', regulator: 'VGCCC' },
  { id: 'au-qld', name: 'Queensland', country: 'Australia', region: 'Asia-Pacific', regulator: 'OLGR' },
  { id: 'nz', name: 'New Zealand', country: 'New Zealand', region: 'Asia-Pacific', regulator: 'DIA' },
  { id: 'in-ga', name: 'Goa', country: 'India', region: 'Asia-Pacific', regulator: 'Goa Govt' },
  { id: 'in-sk', name: 'Sikkim', country: 'India', region: 'Asia-Pacific', regulator: 'Sikkim Govt' },
  { id: 'sg', name: 'Singapore', country: 'Singapore', region: 'Asia-Pacific', regulator: 'GRA' },

  // Africa
  { id: 'za-gp', name: 'Gauteng', country: 'South Africa', region: 'Africa', regulator: 'GGB' },
  { id: 'za-wc', name: 'Western Cape', country: 'South Africa', region: 'Africa', regulator: 'WCGRB' },
  { id: 'za-kzn', name: 'KwaZulu-Natal', country: 'South Africa', region: 'Africa', regulator: 'KZNGBB' },
  { id: 'ke', name: 'Kenya', country: 'Kenya', region: 'Africa', regulator: 'BCLB' },
  { id: 'ng', name: 'Nigeria', country: 'Nigeria', region: 'Africa', regulator: 'NLRC' },
  { id: 'gh', name: 'Ghana', country: 'Ghana', region: 'Africa', regulator: 'GCA' },
  { id: 'tz', name: 'Tanzania', country: 'Tanzania', region: 'Africa', regulator: 'GBT' },
]

/**
 * Returns a hierarchical tree: [{ region, countries: [{ country, jurisdictions }] }]
 */
export function getJurisdictionTree() {
  const regionMap = new Map()
  for (const j of JURISDICTIONS) {
    if (!regionMap.has(j.region)) regionMap.set(j.region, new Map())
    const countryMap = regionMap.get(j.region)
    if (!countryMap.has(j.country)) countryMap.set(j.country, [])
    countryMap.get(j.country).push(j)
  }
  return [...regionMap.entries()].map(([region, countryMap]) => ({
    region,
    countries: [...countryMap.entries()].map(([country, jurisdictions]) => ({
      country,
      jurisdictions,
    })),
  }))
}

/**
 * Returns jurisdiction objects matching the given IDs.
 */
export function getJurisdictionsByIds(ids) {
  const set = new Set(ids)
  return JURISDICTIONS.filter(j => set.has(j.id))
}

/**
 * Returns all jurisdiction IDs within a region.
 */
export function getJurisdictionIdsByRegion(region) {
  return JURISDICTIONS.filter(j => j.region === region).map(j => j.id)
}

/**
 * Returns all jurisdiction IDs within a country.
 */
export function getJurisdictionIdsByCountry(country) {
  return JURISDICTIONS.filter(j => j.country === country).map(j => j.id)
}

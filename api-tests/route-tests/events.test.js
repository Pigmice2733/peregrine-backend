const api = require('./../api.test')
const fetch = require('node-fetch')

test('events', async () => {
  // /events endpoint
  let resp = await fetch(api.address + '/events')
  expect(resp.status).toBe(200)

  let d = await resp.json()

  expect(d).toEqual({ data: expect.any(Array) })
  expect(d.data.length).toBeGreaterThan(1)
  d.data.forEach(event => {
    expect(event.name).toBeA(String)
    expect(event.startDate).toBeADateString()
    expect(event.endDate).toBeADateString()
    expect(event.location).toBeA(Object)
    expect(event.location.lat).toBeA(Number)
    expect(event.location.lon).toBeA(Number)
    expect(event.key).toBeA(String)
    expect(event.district).toBeUndefinedOr(String)
    expect(event.fullDistrict).toBeUndefinedOr(String)
    expect(event.week).toBeUndefinedOr(Number)
    expect(Object.keys(event)).toBeASubsetOf([
      'key',
      'realmId',
      'name',
      'week',
      'startDate',
      'endDate',
      'location',
      'district',
      'fullDistrict',
    ])
  })

  // /events create endpoint
  expect(api.seedUser.roles.isSuperAdmin).toBe(true)

  let event = {
    key: '1970flir',
    name: 'FLIR x Daimler',
    district: 'pnw',
    fullDistrict: 'Pacific Northwest',
    week: 4,
    startDate: '1970-01-01T19:46:40-08:00',
    endDate: '1970-01-02T09:40:00-08:00',
    location: {
      name: 'Cleveland High School',
      lat: 45.498555,
      lon: -122.6385231,
    },
    webcasts: [
      {
        type: 'twitch',
        url: 'https://www.twitch.tv/firstwa_red',
      },
    ],
  }

  resp = await fetch(api.address + '/events', {
    method: 'POST',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(201)

  let respInfo = await fetch(api.address + `/events/${event.key}`, {
    method: 'GET',
    headers: {
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(respInfo.status).toBe(200)

  d = await respInfo.json()

  expect(d.data).toEqual({
    key: event.key,
    realmId: api.seedUser.realmId,
    name: event.name,
    district: event.district,
    fullDistrict: event.fullDistrict,
    week: event.week,
    startDate: expect.toEqualDate(event.startDate),
    endDate: expect.toEqualDate(event.endDate),
    location: event.location,
    webcasts: event.webcasts,
  })

  // /events/{eventKey} endpoint
  resp = await fetch(api.address + '/events/2018flor')
  expect(resp.status).toBe(200)

  d = await resp.json()

  let info = d.data
  expect(info.key).toEqual('2018flor')
  expect(info.name).toBeA(String)
  expect(info.startDate).toBeADateString()
  expect(info.endDate).toBeADateString()
  expect(info.location).toBeA(Object)
  expect(info.location.name).toBeA(String)
  expect(info.location.lat).toBeA(Number)
  expect(info.location.lon).toBeA(Number)
  expect(info.key).toBeA(String)
  expect(info.district).toBeUndefinedOr(String)
  expect(info.fullDistrict).toBeUndefinedOr(String)
  expect(info.week).toBeUndefinedOr(Number)
  expect(info.webcasts).toEqual(expect.any(Array))
  info.webcasts.forEach(webcast => {
    expect(webcast).toEqual({
      url: expect.any(String),
      type: expect.stringMatching(api.youtubeOrTwitch),
    })
  })
  expect(Object.keys(info)).toBeASubsetOf([
    'key',
    'realmId',
    'schemaId',
    'name',
    'week',
    'startDate',
    'endDate',
    'location',
    'district',
    'fullDistrict',
    'webcasts',
  ])
})

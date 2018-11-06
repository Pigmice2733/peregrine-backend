const api = require('./../api.test')
const fetch = require('node-fetch')

test('/events endpoint', async () => {
  const resp = await fetch(api.address + '/events')
  expect(resp.status).toBe(200)

  const d = await resp.json()

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
      'name',
      'week',
      'startDate',
      'endDate',
      'location',
      'district',
      'fullDistrict',
    ])
  })
})

test('/events create endpoint', async () => {
  expect(api.config.seedUser.roles.isAdmin).toBe(true)

  const event = {
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

  const resp = await fetch(api.address + '/events', {
    method: 'PUT',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(201)

  const respInfo = await fetch(api.address + `/events/${event.key}`)
  expect(respInfo.status).toBe(200)

  const d = await respInfo.json()

  expect(d.data).toEqual({
    key: event.key,
    name: event.name,
    district: event.district,
    fullDistrict: event.fullDistrict,
    week: event.week,
    startDate: expect.toEqualDate(event.startDate),
    endDate: expect.toEqualDate(event.endDate),
    location: event.location,
    webcasts: event.webcasts,
  })
})

test('/events/{eventKey} endpoint', async () => {
  const resp = await fetch(api.address + '/events/2018flor')
  expect(resp.status).toBe(200)

  const d = await resp.json()

  const info = d.data
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

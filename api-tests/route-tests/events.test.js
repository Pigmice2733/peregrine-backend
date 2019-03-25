const api = require('./../api.test')
const fetch = require('node-fetch')

test('events', async () => {
  // /events endpoint
  let resp = await fetch(api.address + '/events')
  expect(resp.status).toBe(200)

  let d = await resp.json()

  expect(d).toEqual(expect.any(Array))
  expect(d.length).toBeGreaterThan(1)
  d.forEach(event => expect(event).toBeAnEvent())

  // /events create endpoint
  expect(api.seedUser.roles.isSuperAdmin).toBe(true)

  let event = {
    key: '1970flir' + Number(new Date()),
    name: 'FLIR x Daimler',
    district: 'pnw',
    fullDistrict: 'Pacific Northwest',
    week: 4,
    startDate: '1970-01-01T19:46:40-08:00',
    endDate: '1970-01-02T09:40:00-08:00',
    locationName: 'Cleveland High School',
    lat: 45.498555,
    lon: -122.6385231,
    webcasts: ['https://www.twitch.tv/firstwa_red'],
  }

  resp = await fetch(api.address + `/events/${event.key}`, {
    method: 'PUT',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(201)

  let respInfo = await fetch(api.address + `/events/${event.key}`, {
    method: 'GET',
    headers: {
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(respInfo.status).toBe(200)

  d = await respInfo.json()

  expect(d).toEqual({
    key: event.key,
    realmId: api.seedUser.realmId,
    name: event.name,
    district: event.district,
    fullDistrict: event.fullDistrict,
    week: event.week,
    startDate: expect.toEqualDate(event.startDate),
    endDate: expect.toEqualDate(event.endDate),
    locationName: event.locationName,
    lat: event.lat,
    lon: event.lon,
    webcasts: event.webcasts,
    tbaDeleted: false,
  })

  // /events/{eventKey} endpoint
  resp = await fetch(api.address + '/events/2018flor')
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d).toBeAnEvent()
  expect(d.key).toBe('2018flor')
})

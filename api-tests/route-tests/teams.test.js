const api = require('./../api.test')
const fetch = require('node-fetch')

test('events/{eventKey}/teams', async () => {
  // events/{eventKey}/teams endpoint
  let resp = await fetch(api.address + '/events/2018flor/teams')

  expect(resp.status).toBe(200)
  let d = await resp.json()

  const teams = d
  expect(teams.length).toBeGreaterThan(0)
  expect(teams).toEqual(expect.any(Array))
  expect(teams[0]).toEqual(expect.any(String))

  // /events/{eventKey}/teams/{teamKey} endpoint
  resp = await fetch(api.address + '/events/2018flor/teams/frc1065')
  expect(resp.status).toBe(200)

  d = await resp.json()

  const info = d
  expect(info.rank).toBeUndefinedOr(Number)
  expect(info.rankingScore).toBeUndefinedOr(Number)
  expect(Object.keys(info)).toBeASubsetOf(['rank', 'rankingScore'])

  let date = Number(new Date())

  let event = {
    key: '1966flir' + date,
    name: 'FLIR x Daimler',
    district: 'pnw',
    fullDistrict: 'Pacific Northwest',
    week: 4,
    startDate: '1966-01-01T19:46:40-08:00',
    endDate: '1966-01-02T09:40:00-08:00',
    location: {
      name: 'Cleveland High School',
      lat: 45.498555,
      lon: -122.6385231,
    },
    webcasts: [],
  }

  await fetch(api.address + `/events/${event.key}`, {
    method: 'PUT',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  event = {
    key: '1967flir' + date,
    name: 'FLIR x Daimler',
    district: 'pnw',
    fullDistrict: 'Pacific Northwest',
    week: 4,
    startDate: '1967-01-01T19:46:40-08:00',
    endDate: '1967-01-02T09:40:00-08:00',
    location: {
      name: 'Cleveland High School',
      lat: 45.498555,
      lon: -122.6385231,
    },
    webcasts: [],
  }

  await fetch(api.address + `/events/${event.key}`, {
    method: 'PUT',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  let match = {
    key: 'foo123',
    time: '2018-03-09T11:00:13-08:00',
    redScore: 368,
    blueScore: 74,
    redAlliance: ['frc1592', 'frc5722', 'frc1421'],
    blueAlliance: ['frc6322', 'frc4024', 'frc5283'],
  }

  resp = await fetch(api.address + `/events/1966flir${date}/matches`, {
    method: 'POST',
    body: JSON.stringify(match),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(201)

  resp = await fetch(api.address + `/events/1967flir${date}/matches`, {
    method: 'POST',
    body: JSON.stringify(match),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(201)

  let report = {
    autoName: 'FarScale',
    data: {
      auto: [
        {
          name: 'cross line',
          attempts: 1,
          successes: 1,
        },
        {
          name: 'scale',
          attempts: 2,
          successes: 0,
        },
      ],
      teleop: [
        {
          name: 'exchange',
          attempts: 12,
          successes: 10,
        },
      ],
    },
  }

  resp = await fetch(
    api.address + `/events/1966flir${date}/matches/foo123/reports/frc5283`,
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(201)

  resp = await fetch(
    api.address + `/events/1967flir${date}/matches/foo123/reports/frc5283`,
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(201)

  resp = await fetch(
    api.address + `/events/1966flir${date}/matches/foo123/reports/frc4024`,
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(201)

  resp = await fetch(api.address + '/events/reports/frc5283', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()
  expect(d).toHaveLength(2)
})

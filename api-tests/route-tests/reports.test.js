const api = require('./../api.test')
const fetch = require('node-fetch')

test('reports endpoint', async () => {
  // /matches create endpoint
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
    webcasts: [],
  }

  let eventResp = await fetch(api.address + '/events', {
    method: 'POST',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(eventResp.status).toBe(201)

  let match = {
    key: 'foo123',
    time: '2018-03-09T11:00:13-08:00',
    redScore: 368,
    blueScore: 74,
    redAlliance: ['frc1592', 'frc5722', 'frc1421'],
    blueAlliance: ['frc6322', 'frc4024', 'frc5283'],
  }

  let matchResp = await fetch(api.address + '/events/1970flir/matches', {
    method: 'POST',
    body: JSON.stringify(match),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(matchResp.status).toBe(201)

  let resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
  )
  expect(resp.status).toBe(200)
  let d = await resp.json()

  expect(d.data).toHaveLength(0)

  let report = {
    autoName: 'FarScale',
    auto: [
      {
        statName: 'cross line',
        attempted: true,
        succeeded: true,
      },
      {
        statName: 'scale',
        attempts: 2,
        successes: 0,
      },
    ],
    teleop: [
      {
        statName: 'exchange',
        attempts: 12,
        successes: 10,
      },
    ],
  }

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(report),
    },
  )
  expect(resp.status).toBe(401)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(200)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d.data).toHaveLength(1)
  expect(d.data[0].reporter).toEqual(api.seedUser.username)
  let reporterId = d.data[0].reporterId
  expect(reporterId).not.toBeUndefined()
  expect(d.data[0].autoName).toEqual(report.autoName)
  expect(d.data[0].auto).toEqual(report.auto)
  expect(d.data[0].teleop).toEqual(report.teleop)
  expect(Object.keys(d.data[0])).toBeASubsetOf([
    'reporter',
    'reporterId',
    'autoName',
    'auto',
    'teleop',
  ])

  report.auto = []
  report.autoName = 'NearScale'

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(200)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d.data).toHaveLength(1)
  expect(d.data[0].reporter).toEqual(api.seedUser.username)
  expect(d.data[0].reporterId).toEqual(reporterId)
  expect(d.data[0].autoName).toEqual(report.autoName)
  expect(d.data[0].auto).toEqual(report.auto)
  expect(d.data[0].teleop).toEqual(report.teleop)
  expect(Object.keys(d.data[0])).toBeASubsetOf([
    'reporter',
    'reporterId',
    'autoName',
    'auto',
    'teleop',
  ])
})
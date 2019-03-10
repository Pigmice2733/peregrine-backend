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

  await fetch(api.address + `/events/${event.key}`, {
    method: 'PUT',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

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
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(matchResp.status).toBe(201)

  let resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
  )
  expect(resp.status).toBe(200)
  let d = await resp.json()

  expect(d).toHaveLength(0)

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
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(201)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d).toHaveLength(1)
  let reporterId = d[0].reporterId
  expect(reporterId).not.toBeUndefined()
  expect(d[0].autoName).toEqual(report.autoName)
  expect(d[0].data).not.toBeUndefined()
  expect(d[0].data.auto).toEqual(report.data.auto)
  expect(d[0].data.teleop).toEqual(report.data.teleop)
  expect(Object.keys(d[0])).toBeASubsetOf(['reporterId', 'autoName', 'data'])

  report.auto = []
  report.autoName = 'NearScale'

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(204)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d).toHaveLength(1)
  expect(d[0].reporterId).toEqual(reporterId)
  expect(d[0].autoName).toEqual(report.autoName)
  expect(d[0].data).not.toBeUndefined()
  expect(d[0].data.auto).toEqual(report.data.auto)
  expect(d[0].data.teleop).toEqual(report.data.teleop)
  expect(Object.keys(d[0])).toBeASubsetOf(['reporterId', 'autoName', 'data'])

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d).toEqual([{"autoName": "NearScale", "data": {"auto": [{"attempts": 1, "name": "cross line", "successes": 1}, {"attempts": 2, "name": "scale", "successes": 0}], "teleop": [{"attempts": 12, "name": "exchange", "successes": 10}]}, "reporterId": 1}])
})

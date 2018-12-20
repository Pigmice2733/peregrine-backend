const api = require('./../api.test')
const fetch = require('node-fetch')

test('stats endpoints', async () => {
  // /matches create endpoint
  expect(api.seedUser.roles.isSuperAdmin).toBe(true)

  let schema = {
    year: 1968,
    auto: [
      {
        statName: 'Crossed Line',
        type: 'boolean',
      },
    ],
    teleop: [
      {
        statName: 'Fuel',
        type: 'number',
      },
      {
        statName: 'Cubes',
        type: 'number',
      },
    ],
  }

  let resp = await fetch(api.address + '/schemas', {
    method: 'POST',
    body: JSON.stringify(schema),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(201)

  resp = await fetch(api.address + '/schemas', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  let d = await resp.json()
  let foundSchema = d.data.find(curSchema => schema.year === curSchema.year)

  let event = {
    key: '1968flir',
    name: 'FLIR x Daimler',
    schemaId: foundSchema.id,
    district: 'pnw',
    fullDistrict: 'Pacific Northwest',
    week: 4,
    startDate: '1968-01-01T19:46:40-08:00',
    endDate: '1968-01-02T09:40:00-08:00',
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
      Authorization: 'Bearer ' + (await api.getJWT()),
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

  let matchResp = await fetch(api.address + '/events/1968flir/matches', {
    method: 'POST',
    body: JSON.stringify(match),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(matchResp.status).toBe(201)

  let statsResp = await fetch(api.address + '/events/1968flir/stats', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(statsResp.status).toBe(200)

  d = await statsResp.json()

  let teams = ['frc1592', 'frc5722', 'frc1421', 'frc6322', 'frc4024', 'frc5283']

  d.data.forEach(teamStats => {
    expect(teams).toContain(teamStats.team)
    teams.splice(teams.findIndex(t => t === teamStats.team), 1)
    expect(teamStats.auto).not.toBeUndefined()
    expect(teamStats.teleop).not.toBeUndefined()
  })

  expect(teams).toHaveLength(0)
})

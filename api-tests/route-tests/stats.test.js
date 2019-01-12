const api = require('./../api.test')
const fetch = require('node-fetch')

test('stats endpoints', async () => {
  // /matches create endpoint
  expect(api.seedUser.roles.isSuperAdmin).toBe(true)

  let schema = {
    year: 1968,
    auto: [
      {
        name: 'Crossed Line',
        type: 'boolean',
      },
      {
        name: 'Cubes',
        type: 'number',
      },
    ],
    teleop: [
      {
        name: 'Climbed',
        type: 'boolean',
      },
      {
        name: 'Cubes',
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

  let scout1 = {
    username: 'scout1',
    password: 'password',
    realmId: api.seedUser.realmId,
    firstName: 'test',
    lastName: 'test',
    roles: { isVerified: true, isAdmin: false, isSuperAdmin: false },
  }

  resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(scout1),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status === 201 || resp.status === 409).toBeTruthy()

  let scout2 = {
    username: 'scout2',
    password: 'password',
    realmId: api.seedUser.realmId,
    firstName: 'test',
    lastName: 'test',
    roles: { isVerified: true, isAdmin: false, isSuperAdmin: false },
  }

  resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(scout2),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status === 201 || resp.status === 409).toBeTruthy()

  let realm = {
    name: 'FRC 1592',
    shareReports: false,
  }

  let realmResp = await fetch(api.address + '/realms', {
    method: 'POST',
    body: JSON.stringify(realm),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(realmResp.status).toBe(200)
  d = await realmResp.json()
  realm.id = d.data

  let otherScout = {
    username: 'otherScout',
    password: 'password',
    realmId: realm.id,
    firstName: 'test',
    lastName: 'test',
    roles: { isVerified: true, isAdmin: false, isSuperAdmin: false },
  }

  resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(otherScout),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status === 201 || resp.status === 409).toBeTruthy()

  let firstReport = {
    autoName: 'Cubey',
    data: {
      auto: [
        {
          name: 'Crossed Line',
          attempted: true,
          succeeded: true,
        },
        {
          name: 'Cubes',
          attempts: 2,
          successes: 1,
        },
      ],
      teleop: [
        {
          name: 'Climbed',
          attempted: true,
          succeeded: true,
        },
        {
          name: 'Cubes',
          attempts: 6,
          successes: 7,
        },
      ],
    },
  }

  resp = await fetch(
    api.address + '/events/1968flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(firstReport),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(200)

  let secondReport = {
    autoName: 'Cubey',
    data: {
      auto: [
        {
          name: 'Crossed Line',
          attempted: true,
          succeeded: true,
        },
        {
          name: 'Cubes',
          attempts: 2,
          successes: 2,
        },
      ],
      teleop: [
        {
          name: 'Cubes',
          attempts: 12,
          successes: 10,
        },
      ],
    },
  }

  resp = await fetch(
    api.address + '/events/1968flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(secondReport),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT(scout1)),
      },
    },
  )
  expect(resp.status).toBe(200)

  let report = {
    autoName: 'Cubey',
    data: {
      auto: [
        {
          name: 'Crossed Line',
          attempted: false,
          succeeded: false,
        },
        {
          name: 'Cubes',
          attempts: 5,
          successes: 5,
        },
      ],
      teleop: [
        {
          name: 'Climbed',
          attempted: false,
          succeeded: true,
        },
        {
          name: 'Cubes',
          attempts: 15,
          successes: 15,
        },
      ],
    },
  }

  resp = await fetch(
    api.address + '/events/1968flir/matches/foo123/reports/frc1592',
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(200)

  let thirdReport = {
    autoName: 'Cubey',
    data: {
      auto: [
        {
          name: 'Crossed Line',
          attempted: true,
          succeeded: true,
        },
        {
          name: 'Cubes',
          attempts: 10,
          successes: 10,
        },
      ],
      teleop: [
        {
          name: 'Climbed',
          attempted: true,
          succeeded: true,
        },
        {
          name: 'Cubes',
          attempts: 15,
          successes: 15,
        },
      ],
    },
  }

  resp = await fetch(
    api.address + '/events/1968flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(thirdReport),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT(otherScout)),
      },
    },
  )
  expect(resp.status).toBe(200)

  statsResp = await fetch(api.address + '/events/1968flir/stats', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(statsResp.status).toBe(200)

  d = await statsResp.json()

  teams = ['frc1592', 'frc5722', 'frc1421', 'frc6322', 'frc4024', 'frc5283']

  d.data.forEach(teamStats => {
    expect(teams).toContain(teamStats.team)
    teams.splice(teams.findIndex(t => t === teamStats.team), 1)
    expect(teamStats.auto).not.toBeUndefined()
    expect(teamStats.teleop).not.toBeUndefined()
    if (teamStats.team === 'frc1421') {
      var lineIndex = teamStats.auto[0].name === 'Crossed Line' ? 0 : 1
      expect(teamStats.auto[lineIndex]).toEqual({
        name: 'Crossed Line',
        attempts: 2,
        successes: 2,
      })
      expect(teamStats.auto[1 - lineIndex]).toEqual({
        name: 'Cubes',
        attempts: {
          max: 2,
          avg: 2,
        },
        successes: {
          max: 2,
          avg: 1.5,
        },
      })

      var climbedIndex = teamStats.teleop[0].name === 'Climbed' ? 0 : 1
      expect(teamStats.teleop[climbedIndex]).toEqual({
        name: 'Climbed',
        attempts: 1,
        successes: 1,
      })
      expect(teamStats.teleop[1 - climbedIndex]).toEqual({
        name: 'Cubes',
        attempts: {
          max: 12,
          avg: 9,
        },
        successes: {
          max: 10,
          avg: 8,
        },
      })
    }
  })

  expect(teams).toHaveLength(0)
})

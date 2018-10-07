const fetch = require('node-fetch')
const jsyaml = require('js-yaml')
const fs = require('fs')

expect.extend({
  toBeAnInt(received) {
    const pass = Number.isInteger(received)
    const message = pass
      ? () => `expected ${received} not to be an integer`
      : () => `expected ${received} to be an integer`
    return {
      message,
      pass,
    }
  },
  toBeADateString(received) {
    const parsedDate = new Date(received)
    const pass = !isNaN(Number(parsedDate))
    const message = pass
      ? () => `expected ${received} to not be a valid date string`
      : () => `expected ${received} to be a valid date string`
    return { pass, message }
  },
  toBeA(received, type) {
    try {
      expect(received).toEqual(expect.any(type))
    } catch (error) {
      return { message: error.matcherResult.message, pass: false }
    }
    return { pass: true }
  },
  toBeATeamKey(received) {
    try {
      expect(received).toMatch(/^frc([1-9a-zA-Z])+/)
    } catch (error) {
      return {
        message: () => `expected ${received} to be a team key`,
        pass: false,
      }
    }
    return { pass: true }
  },
  toBeAMatch(received) {
    try {
      expect(received.key).toBeA(String)
      expect(received.time).toBeADateString()
      expect(received.redScore).toBeUndefinedOr(Number)
      expect(received.blueScore).toBeUndefinedOr(Number)
      expect(received.redAlliance).toEqual(expect.any(Array))
      expect(received.redAlliance).toHaveLength(3)
      received.redAlliance.forEach(team => {
        expect(team).toBeATeamKey()
      })
      expect(received.blueAlliance).toEqual(expect.any(Array))
      expect(received.blueAlliance).toHaveLength(3)
      received.blueAlliance.forEach(team => {
        expect(team).toBeATeamKey()
      })
      expect(Object.keys(received)).toBeASubsetOf([
        'key',
        'time',
        'redAlliance',
        'blueAlliance',
        'redScore',
        'blueScore',
      ])
    } catch (error) {
      return {
        message: () => `expected to get a match. failed:\n ${error}`,
        pass: false,
      }
    }
    return { pass: true }
  },
  toBeUndefinedOr(received, type) {
    if (received === undefined) {
      return { pass: true }
    }
    try {
      expect(received).toEqual(expect.any(type))
    } catch (error) {
      return { message: error.matcherResult.message, pass: false }
    }
    return { pass: true }
  },
  toBeASubsetOf(received, items) {
    const s = new Set(items)
    let unexpected = received.reduce(
      (unexpected, i) => (s.has(i) ? unexpected : unexpected.concat(i)),
      [],
    )
    const pass = unexpected.length === 0
    const message = pass
      ? () => `did not expect item(s): ${unexpected}`
      : () => `did not expect item(s): ${unexpected}`
    return { message, pass }
  },
})

const config = jsyaml.safeLoad(
  fs.readFileSync(
    `./../etc/config.${process.env.GO_ENV || 'development'}.yaml`,
    'utf8',
  ),
)

const addr = `http://${config.server.httpAddress}`

const youtubeOrTwitch = /^(youtube|twitch)$/

test('the api is alive', () => {
  return fetch(addr + '/')
})

test('/events endpoint', async () => {
  const d = await fetch(addr + '/events').then(d => d.json())
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
    expect(event.week).toBeUndefinedOr(Number)
    expect(Object.keys(event)).toBeASubsetOf([
      'key',
      'name',
      'week',
      'startDate',
      'endDate',
      'location',
      'district',
    ])
  })
})

test('/events create endpoint', async () => {
  expect(config.seedUser.roles.isAdmin).toBe(true)

  const event = {
    key: '1970flir',
    name: 'FLIR x Daimler',
    district: 'pnw',
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

  const response = await fetch(addr + '/events', {
    method: 'POST',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await getJWT()),
    },
  })

  expect(response.status).toBe(201)

  const d = await fetch(addr + `/events/${event.key}/info`).then(d => d.json())

  expect(d.data).toEqual(event)
})

test('/events/{eventKey}/info endpoint', async () => {
  const d = await fetch(addr + '/events/2018flor/info').then(d => d.json())
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
  expect(info.week).toBeUndefinedOr(Number)
  expect(info.webcasts).toEqual(expect.any(Array))
  info.webcasts.forEach(webcast => {
    expect(webcast).toEqual({
      url: expect.any(String),
      type: expect.stringMatching(youtubeOrTwitch),
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
    'webcasts',
  ])
})

test('/events/{eventKey}/matches endpoint', async () => {
  const d = await fetch(addr + '/events/2018flor/matches').then(d => d.json())
  expect(d).toEqual({ data: expect.any(Array) })
  expect(d.data.length).toBeGreaterThan(1)
  d.data.forEach(match => {
    expect(match).toBeAMatch()
  })
})

test('/matches create endpoint', async () => {
  expect(config.seedUser.roles.isAdmin).toBe(true)

  const match = {
    key: 'foo123',
    predictedTime: '2018-03-09T11:00:13-08:00',
    redScore: 368,
    blueScore: 74,
    redAlliance: ['frc1592', 'frc5722', 'frc1421'],
    blueAlliance: ['frc6322', 'frc4024', 'frc5283'],
  }

  const response = await fetch(addr + '/events/2018flor/matches', {
    method: 'POST',
    body: JSON.stringify(match),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await getJWT()),
    },
  })

  expect(response.status).toBe(201)

  const d = await fetch(
    addr + `/events/2018flor/matches/${match.key}/info`,
  ).then(d => d.json())

  expect(d.data).toStrictEqual({
    key: match.key,
    time: match.predictedTime,
    redScore: match.redScore,
    blueScore: match.blueScore,
    redAlliance: match.redAlliance,
    blueAlliance: match.blueAlliance,
  })
})

test('/events/{eventKey}/matches/{matchKey}/info endpoint', async () => {
  const d = await fetch(addr + '/events/2018flor/matches/qm28/info').then(d =>
    d.json(),
  )
  const info = d.data
  expect(info).toBeAMatch()
})

test('/events/{eventKey}/teams endpoint', async () => {
  const d = await fetch(addr + '/events/2018flor/teams').then(d => d.json())
  const teams = d.data
  expect(teams.length).toBeGreaterThan(0)
  expect(teams).toEqual(expect.any(Array))
  expect(teams[0]).toEqual(expect.any(String))
})

test('/events/{eventKey}/teams/{teamKey}/info endpoint', async () => {
  const d = await fetch(addr + '/events/2018flor/teams/frc1065/info').then(d =>
    d.json(),
  )
  const info = d.data
  expect(info.rank).toBeUndefinedOr(Number)
  expect(info.rankingScore).toBeUndefinedOr(Number)
  expect(info.nextMatch).toBeUndefinedOr(Object)
  if (info.nextMatch !== undefined) {
    expect(info.nextMatch).toBeAMatch()
  }
  expect(Object.keys(info)).toBeASubsetOf(['nextMatch', 'rank', 'rankingScore'])
})

test('/authenticate route', async () => {
  const response = await fetch(addr + '/authenticate', {
    method: 'POST',
    body: JSON.stringify({
      username: config.seedUser.username,
      password: config.seedUser.password,
    }),
    headers: { 'Content-Type': 'application/json' },
  })

  expect(response.status).toBe(200)

  const d = await response.json()
  expect(d.data.jwt).toBeA(String)
})

test('/authenticate route with incorrect auth info', async () => {
  const response = await fetch(addr + '/authenticate', {
    method: 'POST',
    body: JSON.stringify({
      username: config.seedUser.username,
      password: config.seedUser.password + 'a',
    }),
    headers: { 'Content-Type': 'application/json' },
  })

  expect(response.status).toBe(401)
})

const getJWT = async () => {
  const d = await fetch(addr + '/authenticate', {
    method: 'POST',
    body: JSON.stringify({
      username: config.seedUser.username,
      password: config.seedUser.password,
    }),
    headers: { 'Content-Type': 'application/json' },
  }).then(d => d.json())

  return d.data.jwt
}

test('/users create route', async () => {
  const response = await fetch(addr + '/users', {
    method: 'POST',
    body: JSON.stringify({
      username: 'users-create' + Number(new Date()),
      password: 'password',
      firstName: 'test',
      lastName: 'test',
    }),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await getJWT()),
    },
  })

  expect(response.status).toBe(201)
})

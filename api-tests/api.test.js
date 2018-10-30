const fetch = require('node-fetch')
const jsyaml = require('js-yaml')
const fs = require('fs')

expect.extend(require('./matchers.js'))

const config = jsyaml.safeLoad(
  fs.readFileSync(
    `./../etc/config.${process.env.GO_ENV || 'development'}.yaml`,
    'utf8',
  ),
)

const addr = `http://${config.server.httpAddress}`

const youtubeOrTwitch = /^(youtube|twitch)$/

describe('health check endpoints', () => {
  test('the api is listening', () => {
    return fetch(addr + '/')
  })

  test('the api is healthy', async () => {
    const resp = await fetch(addr + '/')
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data.ok).toBe(true)
    expect(d.data.listen.http).toBe(config.server.httpAddress)
    expect(d.data.services.tba).toBe(true)
    expect(d.data.services.postgresql).toBe(true)
  })
})

describe('events endpoints', () => {
  test('/events endpoint', async () => {
    const resp = await fetch(addr + '/events')
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
    expect(config.seedUser.roles.isAdmin).toBe(true)

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

    const resp = await fetch(addr + '/events', {
      method: 'PUT',
      body: JSON.stringify(event),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await getJWT()),
      },
    })

    expect(resp.status).toBe(201)

    const respInfo = await fetch(addr + `/events/${event.key}/info`)
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

  test('/events/{eventKey}/info endpoint', async () => {
    const resp = await fetch(addr + '/events/2018flor/info')
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
      'fullDistrict',
      'webcasts',
    ])
  })
})

describe('match endpoints', () => {
  test('/events/{eventKey}/matches endpoint', async () => {
    const resp = await fetch(addr + '/events/2018flor/matches')
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d).toEqual({ data: expect.any(Array) })
    expect(d.data.length).toBeGreaterThan(1)
    d.data.forEach(match => {
      expect(match).toBeAMatch()
      expect(match.scheduledTime).toBeADateString()
    })
  })

  test('/matches create endpoint', async () => {
    expect(config.seedUser.roles.isAdmin).toBe(true)

    const match = {
      key: 'foo123',
      time: '2018-03-09T11:00:13-08:00',
      redScore: 368,
      blueScore: 74,
      redAlliance: ['frc1592', 'frc5722', 'frc1421'],
      blueAlliance: ['frc6322', 'frc4024', 'frc5283'],
    }

    const resp = await fetch(addr + '/events/2018flor/matches', {
      method: 'PUT',
      body: JSON.stringify(match),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await getJWT()),
      },
    })

    expect(resp.status).toBe(201)

    const respGet = await fetch(
      addr + `/events/2018flor/matches/${match.key}/info`,
    )
    expect(respGet.status).toBe(200)

    const d = await respGet.json()

    expect(d.data).toBeAMatch()
    expect(d.data.scheduledTime).toBeUndefined()
    expect(d.data).toEqual({
      key: match.key,
      time: expect.toEqualDate(match.time),
      redScore: match.redScore,
      blueScore: match.blueScore,
      redAlliance: match.redAlliance,
      blueAlliance: match.blueAlliance,
    })
  })

  test('/events/{eventKey}/matches/{matchKey}/info endpoint', async () => {
    const resp = await fetch(addr + '/events/2018flor/matches/qm28/info')
    expect(resp.status).toBe(200)

    const d = await resp.json()

    const info = d.data
    expect(info.scheduledTime).toBeUndefined()
    expect(info).toBeAMatch()
  })
})

describe('team endpoints', () => {
  test('/events/{eventKey}/teams endpoint', async () => {
    const resp = await fetch(addr + '/events/2018flor/teams')

    expect(resp.status).toBe(200)
    const d = await resp.json()

    const teams = d.data
    expect(teams.length).toBeGreaterThan(0)
    expect(teams).toEqual(expect.any(Array))
    expect(teams[0]).toEqual(expect.any(String))
  })

  test('/events/{eventKey}/teams/{teamKey}/info endpoint', async () => {
    const resp = await fetch(addr + '/events/2018flor/teams/frc1065/info')
    expect(resp.status).toBe(200)

    const d = await resp.json()

    const info = d.data
    expect(info.rank).toBeUndefinedOr(Number)
    expect(info.rankingScore).toBeUndefinedOr(Number)
    expect(info.nextMatch).toBeUndefinedOr(Object)
    if (info.nextMatch !== undefined) {
      expect(info.nextMatch.scheduledTime).toBeUndefined()
      expect(info.nextMatch).toBeAMatch()
    }
    expect(Object.keys(info)).toBeASubsetOf([
      'nextMatch',
      'rank',
      'rankingScore',
    ])
  })
})

describe('auth endpoints', () => {
  test('/authenticate route', async () => {
    const resp = await fetch(addr + '/authenticate', {
      method: 'POST',
      body: JSON.stringify({
        username: config.seedUser.username,
        password: config.seedUser.password,
      }),
      headers: { 'Content-Type': 'application/json' },
    })

    expect(resp.status).toBe(200)

    const d = await resp.json()
    expect(d.data.jwt).toBeA(String)
  })

  test('/authenticate route with incorrect auth info', async () => {
    const resp = await fetch(addr + '/authenticate', {
      method: 'POST',
      body: JSON.stringify({
        username: config.seedUser.username,
        password: config.seedUser.password + 'a',
      }),
      headers: { 'Content-Type': 'application/json' },
    })

    expect(resp.status).toBe(401)
  })
})

const getJWT = async (user = config.seedUser) => {
  const resp = await fetch(addr + '/authenticate', {
    method: 'POST',
    body: JSON.stringify({
      username: user.username,
      password: user.password,
    }),
    headers: { 'Content-Type': 'application/json' },
  })

  const d = await resp.json()

  return d.data.jwt
}

describe('users crud endpoints', () => {
  let user

  test('/users create route', async () => {
    user = {
      username: 'users-create' + Number(new Date()),
      password: 'password',
      firstName: 'test',
      lastName: 'test',
    }

    const resp = await fetch(addr + '/users', {
      method: 'POST',
      body: JSON.stringify(user),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await getJWT()),
      },
    })

    expect(resp.status).toBe(201)
  })

  test('/users get route', async () => {
    const resp = await fetch(addr + '/users')
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data.length).toBeGreaterThanOrEqual(1)

    const foundUser = d.data.find(curUser => curUser.username === user.username)
    expect(foundUser).not.toBe(undefined)

    user = Object.assign(user, foundUser)
  })

  test('/users/{id} get route', async () => {
    const resp = await fetch(addr + '/users/' + user.id)
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data).toEqual({
      id: user.id,
      username: user.username,
      firstName: user.firstName,
      lastName: user.lastName,
      stars: user.stars,
      roles: user.roles,
    })
  })

  test('/users/{id} complete admin patch route', async () => {
    const patchUser = {
      id: user.id,
      username: user.username + 'foo',
      password: user.password + 'b',
      firstName: user.firstName + 'bar',
      lastName: user.lastName + 'foo',
      stars: (user.stars || []).concat('2018flor_qm28'),
      roles: {},
    }
    patchUser.roles.isAdmin = !(user.roles.isAdmin || true)

    const resp = await fetch(addr + '/users/' + patchUser.id, {
      method: 'PATCH',
      body: JSON.stringify(patchUser),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await getJWT()),
      },
    })

    expect(resp.status).toBe(204)

    user = Object.assign(user, patchUser)
  })

  test('/users/{id} partial admin patch route', async () => {
    const patchUser = {
      username: user.username + 'bar',
      roles: { isVerified: true },
    }

    const resp = await fetch(addr + '/users/' + user.id, {
      method: 'PATCH',
      body: JSON.stringify(patchUser),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await getJWT()),
      },
    })

    expect(resp.status).toBe(204)

    user = Object.assign(user, patchUser)
  })

  test('/users/{id} complete self patch route', async () => {
    const patchUser = {
      username: user.username + 'foo',
      password: user.password + 'bla',
      firstName: user.firstName + 'bar',
      lastName: user.lastName + 'foo',
      stars: (user.stars || []).concat('2018flor_qm29'),
    }

    const resp = await fetch(addr + '/users/' + user.id, {
      method: 'PATCH',
      body: JSON.stringify(user),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await getJWT(user)),
      },
    })

    expect(resp.status).toBe(204)

    user = Object.assign(user, patchUser)
  })
})

import fetch from 'node-fetch'
import jsyaml from 'js-yaml'
import fs from 'fs'

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
      expect(received.length).toBeGreaterThan(3)
      expect(received.slice(0, 3)).toEqual('frc')
    } catch (error) {
      return {
        message: () => `expected ${received} to be a team key`,
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
  fs.readFileSync(`./../etc/config.${process.env.GO_ENV}.yaml`, 'utf8'),
)

const addr = `http://${config.server.address}/`

const youtubeOrTwitchRegex = /^(youtube|twitch)$/gm

test('the api is alive', () => {
  return fetch(addr)
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
    expect(webcast.url).toBeA(String)
    expect(webcast.type).toBeA(String)
    expect(webcast.type).toEqual(expect.stringMatching(youtubeOrTwitchRegex))
    expect(Object.keys(webcast)).toBeASubsetOf(['type', 'url'])
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
    expect(match.key).toBeA(String)
    expect(match.time).toBeADateString()
    expect(match.redScore).toBeUndefinedOr(Number)
    expect(match.blueScore).toBeUndefinedOr(Number)
    expect(match.redAlliance).toEqual(expect.any(Array))
    match.redAlliance.forEach(team => {
      expect(team).toBeATeamKey()
    })
    expect(match.blueAlliance).toEqual(expect.any(Array))
    match.blueAlliance.forEach(team => {
      expect(team).toBeATeamKey()
    })
    expect(Object.keys(match)).toBeASubsetOf([
      'key',
      'time',
      'redAlliance',
      'blueAlliance',
      'redScore',
      'blueScore',
    ])
  })
})

test('/events/{eventKey}/matches/{matchKey}/info endpoint', async () => {
  const d = await fetch(
    addr + '/events/2018flor/matches/2018flor_qm28/info',
  ).then(d => d.json())
  const info = d.data
  expect(info.key).toEqual('2018flor_qm28')
  expect(info.time).toBeADateString()
  expect(info.redScore).toBeUndefinedOr(Number)
  expect(info.blueScore).toBeUndefinedOr(Number)
  expect(info.redAlliance).toEqual(expect.any(Array))
  expect(info.redAlliance.length).toEqual(3)
  info.redAlliance.forEach(team => {
    expect(team).toBeATeamKey()
  })
  expect(info.blueAlliance).toEqual(expect.any(Array))
  expect(info.blueAlliance.length).toEqual(3)
  info.blueAlliance.forEach(team => {
    expect(team).toBeATeamKey()
  })
  expect(Object.keys(info)).toBeASubsetOf([
    'key',
    'time',
    'redAlliance',
    'blueAlliance',
    'redScore',
    'blueScore',
  ])
})

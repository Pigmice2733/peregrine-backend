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
  fs.readFileSync('./../etc/config.development.yaml', 'utf8'),
)

const addr = `http://${config.server.address}/`

test('the api is alive', () => {
  return fetch(addr)
})

test('/events endpoint', async () => {
  const d = await fetch(addr + '/events').then(d => d.json())
  expect(d).toEqual({ data: expect.any(Array) })
  expect(d.data.length).toBeGreaterThan(1)
  d.data.forEach(event => {
    expect(event.name).toBeA(String)
    expect(event.startDate).toBeA(String)
    expect(event.endDate).toBeA(String)
    expect(event.location).toBeA(Object)
    expect(event.location.lat).toBeA(Number)
    expect(event.location.lon).toBeA(Number)
    expect(event.id).toBeA(String)
    expect(event.district).toBeUndefinedOr(String)
    expect(event.week).toBeUndefinedOr(Number)
    expect(Object.keys(event)).toBeASubsetOf([
      'id',
      'name',
      'week',
      'startDate',
      'endDate',
      'location',
      'district',
    ])
  })
})

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
      message: message,
      pass: pass,
    }
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
  const firstEvent = d.data[0]

  expect(firstEvent.name).toEqual(expect.any(String))
  expect(firstEvent.startDate).toEqual(expect.any(String))
  expect(firstEvent.endDate).toEqual(expect.any(String))
  expect(firstEvent.location).toEqual(expect.any(Object))
  expect(firstEvent.location.lat).toEqual(expect.any(Number))
  expect(firstEvent.location.lon).toEqual(expect.any(Number))

  const startDate = Number(new Date(firstEvent.startDate))
  const endDate = Number(new Date(firstEvent.endDate))
  expect(startDate).not.toBeNaN()
  expect(endDate).not.toBeNaN()
  expect(startDate).toBeLessThan(endDate)

  if ('district' in firstEvent) {
    expect(firstEvent.district).toEqual(expect.any(String))
  }
  if ('week' in firstEvent) {
    expect(firstEvent.week).toBeAnInt()
  }
})

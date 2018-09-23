import fetch from 'node-fetch'
import jsyaml from 'js-yaml'
import fs from 'fs'

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
  expect(firstEvent).toStrictEqual({
    endDate: expect.any(String),
    location: {
      lat: expect.any(Number),
      lon: expect.any(Number),
    },
    name: expect.any(String),
    startDate: expect.any(String),
    week: expect.any(Number),
  })
  const startDate = Number(new Date(firstEvent.startDate))
  const endDate = Number(new Date(firstEvent.endDate))
  expect(startDate).not.toBeNaN()
  expect(endDate).not.toBeNaN()
  expect(startDate).toBeLessThan(endDate)
})

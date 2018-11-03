const api = require('./../api.test')
const fetch = require('node-fetch')

test('/events/{eventKey}/teams endpoint', async () => {
  const resp = await fetch(api.address + '/events/2018flor/teams')

  expect(resp.status).toBe(200)
  const d = await resp.json()

  const teams = d.data
  expect(teams.length).toBeGreaterThan(0)
  expect(teams).toEqual(expect.any(Array))
  expect(teams[0]).toEqual(expect.any(String))
})

test('/events/{eventKey}/teams/{teamKey} endpoint', async () => {
  const resp = await fetch(api.address + '/events/2018flor/teams/frc1065')
  expect(resp.status).toBe(200)

  const d = await resp.json()

  const info = d.data
  expect(info.rank).toBeUndefinedOr(Number)
  expect(info.rankingScore).toBeUndefinedOr(Number)
  expect(Object.keys(info)).toBeASubsetOf(['rank', 'rankingScore'])
})

const api = require('./../api.test')
const fetch = require('node-fetch')

test('/events/{eventKey}/matches', async () => {
  // /events/{eventKey}/matches
  let resp = await fetch(api.address + '/events/2018flor/matches')
  expect(resp.status).toBe(200)

  let d = await resp.json()

  expect(d).toEqual({ data: expect.any(Array) })
  expect(d.length).toBeGreaterThan(1)
  d.forEach(match => {
    expect(match).toBeAMatch()
    expect(match.scheduledTime).toBeADateString()
  })

  // '/events/{eventKey}/matches endpoint with filter for single team
  resp = await fetch(api.address + '/events/2018flor/matches?team=frc4481')
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d).toEqual({ data: expect.any(Array) })
  expect(d.length).toBeGreaterThan(1)
  d.forEach(match => {
    expect(match).toBeAMatch()
    expect(match.scheduledTime).toBeADateString()
    expect(match).toIncludeTeam('frc4481')
  })

  // /events/{eventKey}/matches endpoint with filter for multiple teams
  resp = await fetch(
    api.address + '/events/2018flor/matches?team=frc4481&team=frc6527',
  )
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d).toEqual({ data: expect.any(Array) })
  expect(d.length).toBeGreaterThan(0)
  d.forEach(match => {
    expect(match).toBeAMatch()
    expect(match.scheduledTime).toBeADateString()
    expect(match).toIncludeTeam('frc4481')
    expect(match).toIncludeTeam('frc6527')
  })

  // /events/{eventKey}/matches endpoint with filter for multiple teams on opposite alliances
  resp = await fetch(
    api.address + '/events/2018flor/matches?team=frc180&team=frc1902',
  )
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d).toEqual({ data: expect.any(Array) })
  expect(d.length).toBeGreaterThan(2)
  d.forEach(match => {
    expect(match).toBeAMatch()
    expect(match.scheduledTime).toBeADateString()
    expect(match).toIncludeTeam('frc180')
    expect(match).toIncludeTeam('frc1902')
  })

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

  resp = await fetch(api.address + '/events/1970flir/matches', {
    method: 'POST',
    body: JSON.stringify(match),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(201)

  let respGet = await fetch(
    api.address + `/events/1970flir/matches/${match.key}`,
  )
  expect(respGet.status).toBe(403)

  respGet = await fetch(api.address + `/events/1970flir/matches/${match.key}`, {
    method: 'GET',
    headers: {
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(respGet.status).toBe(200)

  d = await respGet.json()

  expect(d).toBeAMatch()
  expect(d.scheduledTime).toBeUndefined()
  expect(d).toEqual({
    key: match.key,
    time: expect.toEqualDate(match.time),
    redScore: match.redScore,
    blueScore: match.blueScore,
    redAlliance: match.redAlliance,
    blueAlliance: match.blueAlliance,
  })

  // /events/{eventKey}/matches/{matchKey} endpoint
  resp = await fetch(api.address + '/events/2018flor/matches/qm28')
  expect(resp.status).toBe(200)

  d = await resp.json()

  const info = d
  expect(info.scheduledTime).toBeUndefined()
  expect(info).toBeAMatch()
})

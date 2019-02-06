const api = require('./../api.test')
const fetch = require('node-fetch')

test('reports endpoint', async () => {
  // /matches create endpoint
  expect(api.seedUser.roles.isSuperAdmin).toBe(true)

  const realm = {
    name: 'FRC 666' + Number(new Date()),
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
  expect(realmResp.status).toBe(201)
  let d = await realmResp.json()
  realm.id = d.id

  const otherRealm = {
    name: 'FRC 555' + Number(new Date()),
    shareReports: false,
  }

  realmResp = await fetch(api.address + '/realms', {
    method: 'POST',
    body: JSON.stringify(otherRealm),
    headers: {
      'Content-Type': 'applicatsion/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(realmResp.status).toBe(201)
  d = await realmResp.json()
  otherRealm.id = d.id

  let realmAdmin = {
    username: 'radmin' + Number(new Date()),
    password: 'password',
    realmId: realm.id,
    firstName: 'foo',
    lastName: 'bar',
    roles: { isVerified: true, isAdmin: true, isSuperAdmin: false },
  }

  let resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(realmAdmin),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(201)

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

  await fetch(api.address + `/events/${event.key}`, {
    method: 'PUT',
    body: JSON.stringify(event),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(realmAdmin)),
    },
  })

  let otherRealmAdmin = {
    username: 'oradmin' + Number(new Date()),
    password: 'password',
    realmId: otherRealm.id,
    firstName: 'foo',
    lastName: 'bar',
    roles: { isVerified: true, isAdmin: true, isSuperAdmin: false },
  }

  resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(otherRealmAdmin),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(201)

  let match = {
    key: 'foo123',
    time: '2018-03-09T11:00:13-08:00',
    redScore: 368,
    blueScore: 74,
    redAlliance: ['frc1592', 'frc5722', 'frc1421'],
    blueAlliance: ['frc6322', 'frc4024', 'frc5283'],
  }

  let matchResp = await fetch(api.address + '/events/1970flir/matches', {
    method: 'POST',
    body: JSON.stringify(match),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(matchResp.status).toBe(201)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
      },
    },
  )
  expect(resp.status).toBe(403)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d).toHaveLength(0)

  let report = {
    autoName: 'FarScale',
    data: {
      auto: [
        {
          name: 'cross line',
          attempts: 1,
          successes: 1,
        },
        {
          name: 'scale',
          attempts: 2,
          successes: 0,
        },
      ],
      teleop: [
        {
          name: 'exchange',
          attempts: 12,
          successes: 10,
        },
      ],
    },
  }

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
      },
    },
  )
  expect(resp.status).toBe(403)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(201)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d).toHaveLength(1)
  let reporterId = d[0].reporterId
  expect(reporterId).not.toBeUndefined()
  expect(d[0].autoName).toEqual(report.autoName)
  expect(d[0].data).not.toBeUndefined()
  expect(d[0].data.auto).toEqual(report.data.auto)
  expect(d[0].data.teleop).toEqual(report.data.teleop)
  expect(Object.keys(d[0])).toBeASubsetOf(['reporterId', 'autoName', 'data'])

  report.auto = []
  report.autoName = 'NearScale'

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'PUT',
      body: JSON.stringify(report),
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(204)

  resp = await fetch(
    api.address + '/events/1970flir/matches/foo123/reports/frc1421',
    {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT()),
      },
    },
  )
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d).toHaveLength(1)
  expect(d[0].reporterId).toEqual(reporterId)
  expect(d[0].autoName).toEqual(report.autoName)
  expect(d[0].data).not.toBeUndefined()
  expect(d[0].data.auto).toEqual(report.data.auto)
  expect(d[0].data.teleop).toEqual(report.data.teleop)
  expect(Object.keys(d[0])).toBeASubsetOf(['reporterId', 'autoName', 'data'])
})

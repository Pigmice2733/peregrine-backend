const api = require('./../api.test')
const fetch = require('node-fetch')

const realm = {
  name: 'FRC 1234' + Number(new Date()),
  shareReports: false,
}

let realmAdmin

test('realms', async () => {
  // /realms endpoint

  // /realms create unauthorized
  let realmResp = await fetch(api.address + '/realms', {
    method: 'POST',
    body: JSON.stringify(realm),
  })
  expect(realmResp.status).toBe(401)

  // /realms create
  realmResp = await fetch(api.address + '/realms', {
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

  // /realms/{id} update
  realm.name += "foobar"
  realmResp = await fetch(api.address + `/realms/${realm.id}`, {
    method: 'POST',
    body: JSON.stringify(realm),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(realmResp.status).toBe(204)

  realmAdmin = {
    username: 'realmadmin' + Number(new Date()),
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

  expect(realmAdmin.roles.isAdmin).toEqual(true)
  expect(realmAdmin.roles.isSuperAdmin).toEqual(false)
  expect(realmAdmin.roles.isVerified).toEqual(true)
  expect(realmAdmin.realmId).toEqual(realm.id)

  // /realms create unathorized
  realmResp = await fetch(api.address + '/realms', {
    method: 'POST',
    body: JSON.stringify(realm),
  })
  expect(realmResp.status).toBe(401)

  // /realms get super-admin
  resp = await fetch(api.address + '/realms', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.length).toBeGreaterThanOrEqual(1)
  let foundRealm = d.find(curRealm => curRealm.id === realm.id)

  expect(foundRealm).toEqual({
    id: realm.id,
    name: realm.name,
    shareReports: realm.shareReports,
  })
  expect(Object.keys(foundRealm)).toEqual(['id', 'name', 'shareReports'])

  // /realms get no login
  resp = await fetch(api.address + '/realms', {
    method: 'GET',
  })
  expect(resp.status).toBe(200)
  d = await resp.json()
  expect(d).toHaveLength(0)

  // /realms/{id} endpoint
  // /realms/{id} get super-admin
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d).toEqual({
    id: realm.id,
    name: realm.name,
    shareReports: realm.shareReports,
  })
  expect(Object.keys(d)).toEqual(['id', 'name', 'shareReports'])

  // /realms/{id} get admin
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(realmAdmin)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d).toEqual({
    id: realm.id,
    name: realm.name,
    shareReports: realm.shareReports,
  })
  expect(Object.keys(d)).toEqual(['id', 'name', 'shareReports'])

  // /realms/{id} get unauthorized
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'GET',
  })
  expect(resp.status).toBe(403)

  // /realms/{id} delete unauthorized
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'DELETE',
  })

  expect(resp.status).toBe(401)

  // /realms/{id} delete authorized
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(204)

  // test that deletes succeeded
  resp = await fetch(api.address + '/realms', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()
  foundRealm = d.find(curRealm => curRealm.id === realm.id)

  expect(foundRealm).toBeUndefined()
})

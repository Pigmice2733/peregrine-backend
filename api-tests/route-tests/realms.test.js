const api = require('./../api.test')
const fetch = require('node-fetch')

const realm = {
  name: 'FRC 1234',
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
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(realmResp.status).toBe(200)
  let d = await realmResp.json()
  realm.id = d.data

  realmAdmin = {
    username: 'realm-admin',
    password: 'password',
    realmID: realm.id,
    firstName: 'foo',
    lastName: 'bar',
    roles: { isVerified: true, isAdmin: true, isSuperAdmin: false },
  }

  let resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(realmAdmin),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(201)

  expect(realmAdmin.roles.isAdmin).toEqual(true)
  expect(realmAdmin.roles.isSuperAdmin).toEqual(false)
  expect(realmAdmin.roles.isVerified).toEqual(true)
  expect(realmAdmin.realmID).toEqual(realm.id)

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
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data.length).toBeGreaterThanOrEqual(1)
  let foundRealm = d.data.find(curRealm => curRealm.id === realm.id)

  expect(foundRealm).toEqual({
    id: realm.id,
    name: realm.name,
    shareReports: realm.shareReports,
  })
  expect(Object.keys(foundRealm)).toEqual(['id', 'name', 'shareReports'])

  // /realms get unauthorized
  resp = await fetch(api.address + '/realms', {
    method: 'GET',
  })
  expect(resp.status).toBe(401)

  // /realms/{id} endpoint
  // /realms/{id} get super-admin
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    id: realm.id,
    name: realm.name,
    shareReports: realm.shareReports,
  })
  expect(Object.keys(d.data)).toEqual(['id', 'name', 'shareReports'])

  // /realms/{id} get admin
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    id: realm.id,
    name: realm.name,
    shareReports: realm.shareReports,
  })
  expect(Object.keys(d.data)).toEqual(['id', 'name', 'shareReports'])

  // /realms/{id} get unauthorized
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'GET',
  })
  expect(resp.status).toBe(401)

  // /realms/{id} patch unauthorized
  let patchRealm = {
    name: 'Fake',
  }

  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'PATCH',
    body: JSON.stringify(patchRealm),
  })

  expect(resp.status).toBe(401)

  // /realms/{id} patch non-existent
  patchRealm = {
    id: 'blah',
    name: 'Real',
  }

  resp = await fetch(api.address + '/realms/-3', {
    method: 'PATCH',
    body: JSON.stringify(patchRealm),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(404)

  // /realms/{id} complete patch
  patchRealm = {
    id: 'Name',
    shareReports: !realm.shareReports,
  }

  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'PATCH',
    body: JSON.stringify(patchRealm),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })

  realm.name = 'Name'
  realm.shareReports = !realm.shareReports

  expect(resp.status).toBe(204)

  // /realms/{id} partial patch
  patchRealm = {
    id: 'blah',
    name: 'Real',
  }

  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'PATCH',
    body: JSON.stringify(patchRealm),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
    },
  })

  realm.name = 'Real'

  expect(resp.status).toBe(204)

  // check that patches succeeded
  resp = await fetch(api.address + '/realms/' + realm.id, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    id: realm.id,
    name: realm.name,
    shareReports: realm.shareReports,
  })
  expect(Object.keys(d.data)).toEqual(['id', 'name', 'shareReports'])

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
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(204)

  // test that deletes succeeded
  resp = await fetch(api.address + '/realms', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()
  foundRealm = d.data.find(curRealm => curRealm.id === realm.id)

  expect(foundRealm).toBeUndefined()
})
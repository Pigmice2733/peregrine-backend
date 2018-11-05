const api = require('./../api.test')
const fetch = require('node-fetch')

const realm = {
  team: 'frc1234',
  name: 'Numb',
  publicData: false,
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
  realmAdmin = d.data

  expect(realmAdmin.roles.isAdmin).toEqual(true)
  expect(realmAdmin.roles.isSuperAdmin).toEqual(false)
  expect(realmAdmin.roles.isVerified).toEqual(true)
  expect(realmAdmin.realm).toEqual(realm.team)

  // /realms create unathorized
  realmResp = await fetch(api.address + '/realms', {
    method: 'POST',
    body: JSON.stringify(realm),
  })
  expect(realmResp.status).toBe(401)

  // /realms get super-admin
  let resp = await fetch(api.address + '/realms', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data.length).toBeGreaterThanOrEqual(1)
  let foundRealm = d.data.find(curRealm => curRealm.team === realm.team)

  expect(foundRealm).toEqual({
    team: realm.team,
    name: realm.name,
    publicData: realm.publicData,
  })
  expect(Object.keys(foundRealm)).toEqual(['team', 'name', 'publicData'])

  // /realms get unauthorized
  resp = await fetch(api.address + '/realms', {
    method: 'GET',
  })
  expect(resp.status).toBe(401)

  // /realms/{teamKey} endpoint
  // /realms/{teamKey} get super-admin
  resp = await fetch(api.address + '/realms/' + realm.team, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    team: realm.team,
    name: realm.name,
    publicData: realm.publicData,
  })
  expect(Object.keys(d.data)).toEqual(['team', 'name', 'publicData'])

  // /realms/{teamKey} get admin
  resp = await fetch(api.address + '/realms/' + realm.team, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    team: realm.team,
    name: realm.name,
    publicData: realm.publicData,
  })
  expect(Object.keys(d.data)).toEqual(['team', 'name', 'publicData'])

  // /realms/{teamKey} get unauthorized
  resp = await fetch(api.address + '/realms/' + realm.team, {
    method: 'GET',
  })
  expect(resp.status).toBe(401)

  // /realms/{teamKey} patch unauthorized
  let patchRealm = {
    name: 'Fake',
  }

  resp = await fetch(api.address + '/realms/' + realm.team, {
    method: 'PATCH',
    body: JSON.stringify(patchRealm),
  })

  expect(resp.status).toBe(401)

  // /realms/{teamKey} patch non-existent
  patchRealm = {
    team: 'blah',
    name: 'Real',
  }

  resp = await fetch(api.address + '/realms/very_non_existent_and_fake', {
    method: 'PATCH',
    body: JSON.stringify(patchRealm),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(404)

  // /realms/{teamKey} complete patch
  patchRealm = {
    name: 'Name',
    publicData: !realm.publicData,
  }

  resp = await fetch(api.address + '/realms/' + realm.team, {
    method: 'PATCH',
    body: JSON.stringify(patchRealm),
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT()),
    },
  })

  realm.name = 'Name'
  realm.publicData = !realm.publicData

  expect(resp.status).toBe(204)

  // /realms/{teamKey} partial patch
  patchRealm = {
    team: 'blah',
    name: 'Real',
  }

  resp = await fetch(api.address + '/realms/' + realm.team, {
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
  resp = await fetch(api.address + '/realms/' + realm.team, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    team: realm.team,
    name: realm.name,
    publicData: realm.publicData,
  })
  expect(Object.keys(d.data)).toEqual(['team', 'name', 'publicData'])

  // /realms/{teamKey} delete unauthorized
  resp = await fetch(api.address + '/realms/' + realm.team, {
    method: 'DELETE',
  })

  expect(resp.status).toBe(401)

  // /realms/{teamKey} delete authorized
  resp = await fetch(api.address + '/realms/' + realm.team, {
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
  foundRealm = d.data.find(curRealm => curRealm.team === realm.team)

  expect(foundRealm).toBeUndefined()
})

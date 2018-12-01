const api = require('./../api.test')
const fetch = require('node-fetch')

describe('auth endpoints', () => {
  test('/authenticate route', async () => {
    const resp = await fetch(api.address + '/authenticate', {
      method: 'POST',
      body: JSON.stringify({
        username: api.seedUser.username,
        password: api.seedUser.password,
      }),
      headers: { 'Content-Type': 'application/json' },
    })

    expect(resp.status).toBe(200)

    const d = await resp.json()
    expect(d.data.jwt).toBeA(String)
  })

  test('/authenticate route with incorrect auth info', async () => {
    const resp = await fetch(api.address + '/authenticate', {
      method: 'POST',
      body: JSON.stringify({
        username: api.seedUser.username,
        password: api.seedUser.password + 'a',
      }),
      headers: { 'Content-Type': 'application/json' },
    })

    expect(resp.status).toBe(401)
  })
})

test('users CRUD', async () => {
  let sameRealmUser
  let otherRealmUser
  let otherRealmAdmin
  let unverifiedSuperAdmin
  let otherRealmId

  let otherRealm = {
    name: 'TMM' + Number(new Date()),
    shareReports: false,
  }

  // users create
  // /users create non-admin for same realm
  sameRealmUser = {
    username: 'users-create' + Number(new Date()),
    password: 'password',
    realmId: 1,
    firstName: 'test',
    lastName: 'test',
    roles: { isVerified: true },
  }

  let resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(sameRealmUser),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(201)

  // /users create unverified non-admin for different realm
  const realmResp = await fetch(api.address + '/realms', {
    method: 'POST',
    body: JSON.stringify(otherRealm),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(realmResp.status).toBe(200)
  let d = await realmResp.json()
  otherRealmId = d.data

  otherRealmAdmin = {
    username: 'users-other-admin',
    password: 'password',
    realmId: otherRealmId,
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

  otherRealmUser = {
    username: 'users-other-user',
    password: 'password',
    realmId: otherRealmId,
    firstName: 'test',
    lastName: 'test',
    roles: { isVerified: true, isAdmin: true, isSuperAdmin: true },
  }

  resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(otherRealmUser),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(sameRealmUser)),
    },
  })

  expect(resp.status).toBe(201)

  // /users create unverified super-admin
  unverifiedSuperAdmin = {
    username: 'users-super',
    password: 'password',
    realmId: 1,
    firstName: 'test',
    lastName: 'test',
    roles: { isAdmin: false, isSuperAdmin: true, isVerified: true },
  }

  resp = await fetch(api.address + '/users', {
    method: 'POST',
    body: JSON.stringify(unverifiedSuperAdmin),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
    },
  })

  expect(resp.status).toBe(201)

  // users get
  // /users get route super-admin
  resp = await fetch(api.address + '/users', {
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data.length).toBeGreaterThanOrEqual(5)

  let foundUser = d.data.find(
    curUser => curUser.username === sameRealmUser.username,
  )
  expect(foundUser).not.toBe(undefined)
  sameRealmUser = Object.assign(sameRealmUser, foundUser)

  let foundAdmin = d.data.find(
    curUser => curUser.username === unverifiedSuperAdmin.username,
  )
  expect(foundAdmin).not.toBe(undefined)
  unverifiedSuperAdmin = Object.assign(unverifiedSuperAdmin, foundAdmin)

  // /users get route other-realm
  resp = await fetch(api.address + '/users', {
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toHaveLength(2)

  foundAdmin = d.data.find(
    curUser => curUser.username === otherRealmAdmin.username,
  )
  expect(foundAdmin).not.toBe(undefined)
  otherRealmAdmin = Object.assign(otherRealmAdmin, foundAdmin)

  foundUser = d.data.find(
    curUser => curUser.username === otherRealmUser.username,
  )
  expect(foundUser).not.toBe(undefined)
  otherRealmUser = Object.assign(otherRealmUser, foundUser)
  // Assert that otherRealmUser's permissions were created as expected
  expect(otherRealmUser.roles).toEqual({
    isVerified: false,
    isSuperAdmin: false,
    isAdmin: false,
  })

  // /users get route unverified non-admin
  resp = await fetch(api.address + '/users', {
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(otherRealmUser)),
    },
  })
  expect(resp.status).toBe(403)

  // /users get route unverified super-admin', async () => {
  resp = await fetch(api.address + '/users', {
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(unverifiedSuperAdmin)),
    },
  })
  expect(resp.status).toBe(403)

  // /users get route unauthorized', async () => {
  resp = await fetch(api.address + '/users')
  expect(resp.status).toBe(401)

  // /users/{id} get
  // /users/{id} get route
  resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    id: sameRealmUser.id,
    username: sameRealmUser.username,
    realmId: sameRealmUser.realmId,
    firstName: sameRealmUser.firstName,
    lastName: sameRealmUser.lastName,
    stars: sameRealmUser.stars,
    roles: sameRealmUser.roles,
  })

  // /users/{id} get route unauthorized
  resp = await fetch(api.address + '/users/' + sameRealmUser.id)

  expect(resp.status).toBe(401)

  // /users/{id} complete admin patch route
  let patchUser = {
    id: sameRealmUser.id,
    username: sameRealmUser.username + 'foo',
    password: sameRealmUser.password + 'b',
    firstName: sameRealmUser.firstName + 'bar',
    lastName: sameRealmUser.lastName + 'foofah',
    stars: (sameRealmUser.stars || []).concat('2018flor'),
    roles: { isAdmin: false },
  }

  resp = await fetch(api.address + '/users/' + patchUser.id, {
    method: 'PATCH',
    body: JSON.stringify(patchUser),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(204)

  sameRealmUser = Object.assign(sameRealmUser, patchUser)

  // /users/{id} admin patch non-existent
  patchUser = {
    id: sameRealmUser.id,
    username: sameRealmUser.username + 'foo',
    password: sameRealmUser.password + 'b',
    firstName: sameRealmUser.firstName + 'bar',
    lastName: sameRealmUser.lastName + 'foofah',
    stars: (sameRealmUser.stars || []).concat('2018flor'),
    roles: { isAdmin: false },
  }

  resp = await fetch(api.address + '/users/666', {
    method: 'PATCH',
    body: JSON.stringify(patchUser),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(404)

  // /users/{id} partial admin patch route
  patchUser = {
    password: sameRealmUser.password + 'turducken',
    roles: { isVerified: true },
  }

  resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
    method: 'PATCH',
    body: JSON.stringify(patchUser),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })

  expect(resp.status).toBe(204)

  sameRealmUser = Object.assign(sameRealmUser, patchUser)

  // /users/{id} get self route
  resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(sameRealmUser)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    id: sameRealmUser.id,
    username: sameRealmUser.username,
    realmId: sameRealmUser.realmId,
    firstName: sameRealmUser.firstName,
    lastName: sameRealmUser.lastName,
    stars: sameRealmUser.stars,
    roles: { isAdmin: false, isSuperAdmin: false, isVerified: true },
  })

  // /users/{id} bad self patch route
  patchUser = {
    stars: (sameRealmUser.stars || []).concat('2018flor_qm29'),
  }

  resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
    method: 'PATCH',
    body: JSON.stringify(patchUser),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(sameRealmUser)),
    },
  })

  expect(resp.status).toBe(422)

  // /users/{id} complete self patch route
  patchUser = {
    username: sameRealmUser.username + 'foo',
    password: sameRealmUser.password + 'bla',
    firstName: 'spinning',
    lastName: 'yarn',
    stars: (sameRealmUser.stars || []).concat('2018nytv'),
    roles: { isVerified: true, isAdmin: true, isSuperAdmin: true },
  }

  resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
    method: 'PATCH',
    body: JSON.stringify(patchUser),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(sameRealmUser)),
    },
  })

  expect(resp.status).toBe(204)

  sameRealmUser = Object.assign(sameRealmUser, patchUser)

  // test complete self patch succeeded
  resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(sameRealmUser)),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  expect(d.data).toEqual({
    id: sameRealmUser.id,
    username: sameRealmUser.username,
    realmId: sameRealmUser.realmId,
    firstName: sameRealmUser.firstName,
    lastName: sameRealmUser.lastName,
    stars: sameRealmUser.stars,
    roles: { isAdmin: false, isSuperAdmin: false, isVerified: true },
  })

  // '/users/{id} delete
  // /users/{id} delete other realm foridden
  resp = await fetch(api.address + '/users/' + otherRealmUser.id, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(sameRealmUser)),
    },
  })
  expect(resp.status).toBe(403)

  // /users/{id} delete admin forbidden
  resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
    },
  })
  expect(resp.status).toBe(403)

  // /users/{id} delete self
  let respUser = await fetch(api.address + '/users/' + sameRealmUser.id, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(sameRealmUser)),
    },
  })
  expect(respUser.status).toBe(204)

  let respAdmin = await fetch(
    api.address + '/users/' + unverifiedSuperAdmin.id,
    {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        Authorization: 'Bearer ' + (await api.getJWT(unverifiedSuperAdmin)),
      },
    },
  )
  expect(respAdmin.status).toBe(204)

  // /users/{id} delete same realm user
  resp = await fetch(api.address + '/users/' + otherRealmUser.id, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
    },
  })
  expect(resp.status).toBe(204)

  // /users/{id} delete other realm user
  resp = await fetch(api.address + '/users/' + otherRealmAdmin.id, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(204)

  // test that deletes succeeded
  resp = await fetch(api.address + '/users', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  d = await resp.json()

  let deletedUsernames = [
    sameRealmUser.username,
    otherRealmUser.username,
    otherRealmAdmin.username,
  ]
  d.data.forEach(user => {
    expect(deletedUsernames).not.toContain(user.username)
  })
})

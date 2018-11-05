const api = require('./../api.test')
const fetch = require('node-fetch')

describe('auth endpoints', () => {
  test('/authenticate route', async () => {
    const resp = await fetch(api.address + '/authenticate', {
      method: 'POST',
      body: JSON.stringify({
        username: api.config.seedUser.username,
        password: api.config.seedUser.password,
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
        username: api.config.seedUser.username,
        password: api.config.seedUser.password + 'a',
      }),
      headers: { 'Content-Type': 'application/json' },
    })

    expect(resp.status).toBe(401)
  })
})

describe('users crud endpoints', () => {
  let sameRealmUser
  let otherRealmUser
  let otherRealmAdmin
  let unverifiedSuperAdmin

  let otherRealm = {
    team: 'frc2471',
    name: 'TMM',
    publicData: false,
  }

  describe('users create', () => {
    test('/users create non-admin for same realm', async () => {
      sameRealmUser = {
        username: 'users-create' + Number(new Date()),
        password: 'password',
        realm: 'frc2733',
        firstName: 'test',
        lastName: 'test',
        roles: { isVerified: true },
      }

      const resp = await fetch(api.address + '/users', {
        method: 'POST',
        body: JSON.stringify(sameRealmUser),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })

      expect(resp.status).toBe(201)
    })

    test('/users create unverified non-admin for different realm', async () => {
      const realmResp = await fetch(api.address + '/realms', {
        method: 'POST',
        body: JSON.stringify(otherRealm),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })
      expect(realmResp.status).toBe(200)
      const d = await realmResp.json()
      otherRealmAdmin = d.data

      otherRealmUser = {
        username: 'users-other-user',
        password: 'password',
        realm: 'frc2471',
        firstName: 'test',
        lastName: 'test',
        roles: { isVerified: true, isAdmin: true, isSuperAdmin: true },
      }

      const resp = await fetch(api.address + '/users', {
        method: 'POST',
        body: JSON.stringify(otherRealmUser),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(sameRealmUser)),
        },
      })

      expect(resp.status).toBe(201)
    })

    test('/users create unverified super-admin', async () => {
      unverifiedSuperAdmin = {
        username: 'users-super',
        password: 'password',
        realm: 'frc2733',
        firstName: 'test',
        lastName: 'test',
        roles: { isAdmin: false, isSuperAdmin: true, isVerified: true },
      }

      const resp = await fetch(api.address + '/users', {
        method: 'POST',
        body: JSON.stringify(unverifiedSuperAdmin),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
        },
      })

      expect(resp.status).toBe(201)
    })
  })

  describe('users get', () => {
    test('/users get route super-admin', async () => {
      const resp = await fetch(api.address + '/users', {
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })
      expect(resp.status).toBe(200)

      const d = await resp.json()

      expect(d.data.length).toBeGreaterThanOrEqual(5)

      const foundUser = d.data.find(
        curUser => curUser.username === sameRealmUser.username,
      )
      expect(foundUser).not.toBe(undefined)
      sameRealmUser = Object.assign(sameRealmUser, foundUser)

      const foundAdmin = d.data.find(
        curUser => curUser.username === unverifiedSuperAdmin.username,
      )
      expect(foundAdmin).not.toBe(undefined)
      unverifiedSuperAdmin = Object.assign(unverifiedSuperAdmin, foundAdmin)
    })

    test('/users get route other-realm', async () => {
      const resp = await fetch(api.address + '/users', {
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
        },
      })
      expect(resp.status).toBe(200)

      const d = await resp.json()

      expect(d.data).toHaveLength(2)

      const foundAdmin = d.data.find(
        curUser => curUser.username === otherRealmAdmin.username,
      )
      expect(foundAdmin).not.toBe(undefined)
      otherRealmAdmin = Object.assign(otherRealmAdmin, foundAdmin)

      const foundUser = d.data.find(
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
    })

    test('/users get route unverified non-admin', async () => {
      const resp = await fetch(api.address + '/users', {
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(otherRealmUser)),
        },
      })
      expect(resp.status).toBe(403)
    })

    test('/users get route unverified super-admin', async () => {
      const resp = await fetch(api.address + '/users', {
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(unverifiedSuperAdmin)),
        },
      })
      expect(resp.status).toBe(403)
    })

    test('/users get route unauthorized', async () => {
      const resp = await fetch(api.address + '/users')
      expect(resp.status).toBe(401)
    })
  })

  describe('/users/{id} get', () => {
    test('/users/{id} get route', async () => {
      const resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })
      expect(resp.status).toBe(200)

      const d = await resp.json()

      expect(d.data).toEqual({
        id: sameRealmUser.id,
        username: sameRealmUser.username,
        realm: sameRealmUser.realm,
        firstName: sameRealmUser.firstName,
        lastName: sameRealmUser.lastName,
        stars: sameRealmUser.stars,
        roles: sameRealmUser.roles,
      })
    })

    test('/users/{id} get route unauthorized', async () => {
      const resp = await fetch(api.address + '/users/' + sameRealmUser.id)

      expect(resp.status).toBe(401)
    })

    test('/users/{id} complete admin patch route', async () => {
      const patchUser = {
        id: sameRealmUser.id,
        username: sameRealmUser.username + 'foo',
        password: sameRealmUser.password + 'b',
        firstName: sameRealmUser.firstName + 'bar',
        lastName: sameRealmUser.lastName + 'foofah',
        stars: (sameRealmUser.stars || []).concat('2018flor'),
        roles: { isAdmin: false },
      }

      const resp = await fetch(api.address + '/users/' + patchUser.id, {
        method: 'PATCH',
        body: JSON.stringify(patchUser),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })

      expect(resp.status).toBe(204)

      sameRealmUser = Object.assign(sameRealmUser, patchUser)
    })

    test('/users/{id} admin patch non-existent', async () => {
      const patchUser = {
        id: sameRealmUser.id,
        username: sameRealmUser.username + 'foo',
        password: sameRealmUser.password + 'b',
        firstName: sameRealmUser.firstName + 'bar',
        lastName: sameRealmUser.lastName + 'foofah',
        stars: (sameRealmUser.stars || []).concat('2018flor'),
        roles: { isAdmin: false },
      }

      const resp = await fetch(api.address + '/users/666', {
        method: 'PATCH',
        body: JSON.stringify(patchUser),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })

      expect(resp.status).toBe(404)
    })

    test('/users/{id} partial admin patch route', async () => {
      const patchUser = {
        password: sameRealmUser.password + 'turducken',
        roles: { isVerified: true },
      }

      const resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
        method: 'PATCH',
        body: JSON.stringify(patchUser),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })

      expect(resp.status).toBe(204)

      sameRealmUser = Object.assign(sameRealmUser, patchUser)
    })

    test('/users/{id} get self route', async () => {
      const resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(sameRealmUser)),
        },
      })
      expect(resp.status).toBe(200)

      const d = await resp.json()

      expect(d.data).toEqual({
        id: sameRealmUser.id,
        username: sameRealmUser.username,
        realm: sameRealmUser.realm,
        firstName: sameRealmUser.firstName,
        lastName: sameRealmUser.lastName,
        stars: sameRealmUser.stars,
        roles: { isAdmin: false, isSuperAdmin: false, isVerified: true },
      })
    })

    test('/users/{id} bad self patch route', async () => {
      const patchUser = {
        stars: (sameRealmUser.stars || []).concat('2018flor_qm29'),
      }

      const resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
        method: 'PATCH',
        body: JSON.stringify(patchUser),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(sameRealmUser)),
        },
      })

      expect(resp.status).toBe(422)
    })

    test('/users/{id} complete self patch route', async () => {
      const patchUser = {
        username: sameRealmUser.username + 'foo',
        password: sameRealmUser.password + 'bla',
        firstName: 'spinning',
        lastName: 'yarn',
        stars: (sameRealmUser.stars || []).concat('2018nytv'),
        roles: { isVerified: true, isAdmin: true, isSuperAdmin: true },
      }

      const resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
        method: 'PATCH',
        body: JSON.stringify(patchUser),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(sameRealmUser)),
        },
      })

      expect(resp.status).toBe(204)

      sameRealmUser = Object.assign(sameRealmUser, patchUser)
    })

    test('test complete self patch succeeded', async () => {
      const resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(sameRealmUser)),
        },
      })
      expect(resp.status).toBe(200)

      const d = await resp.json()

      expect(d.data).toEqual({
        id: sameRealmUser.id,
        username: sameRealmUser.username,
        realm: sameRealmUser.realm,
        firstName: sameRealmUser.firstName,
        lastName: sameRealmUser.lastName,
        stars: sameRealmUser.stars,
        roles: { isAdmin: false, isSuperAdmin: false, isVerified: true },
      })
    })
  })

  describe('/users/{id} delete', () => {
    test('/users/{id} delete other realm foridden', async () => {
      const resp = await fetch(api.address + '/users/' + otherRealmUser.id, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(sameRealmUser)),
        },
      })
      expect(resp.status).toBe(403)
    })

    test('/users/{id} delete admin forbidden', async () => {
      const resp = await fetch(api.address + '/users/' + sameRealmUser.id, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
        },
      })
      expect(resp.status).toBe(403)
    })

    test('/users/{id} delete self', async () => {
      const respUser = await fetch(api.address + '/users/' + sameRealmUser.id, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(sameRealmUser)),
        },
      })
      expect(respUser.status).toBe(204)

      const respAdmin = await fetch(
        api.address + '/users/' + unverifiedSuperAdmin.id,
        {
          method: 'DELETE',
          headers: {
            'Content-Type': 'application/json',
            Authentication:
              'Bearer ' + (await api.getJWT(unverifiedSuperAdmin)),
          },
        },
      )
      expect(respAdmin.status).toBe(204)
    })

    test('/users/{id} delete same realm user', async () => {
      const resp = await fetch(api.address + '/users/' + otherRealmUser.id, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT(otherRealmAdmin)),
        },
      })
      expect(resp.status).toBe(204)
    })

    test('/users/{id} delete other realm user', async () => {
      const resp = await fetch(api.address + '/users/' + otherRealmAdmin.id, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })
      expect(resp.status).toBe(204)
    })

    test('test that deletes succeeded', async () => {
      const resp = await fetch(api.address + '/users', {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      })
      expect(resp.status).toBe(200)

      const d = await resp.json()

      const deletedUsernames = [
        sameRealmUser.username,
        otherRealmUser.username,
        otherRealmAdmin.username,
      ]
      d.data.forEach(user => {
        expect(deletedUsernames).not.toContain(user.username)
      })
    })
  })
})

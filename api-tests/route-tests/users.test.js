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
  let user

  test('/users create route', async () => {
    user = {
      username: 'users-create' + Number(new Date()),
      password: 'password',
      firstName: 'test',
      lastName: 'test',
    }

    const resp = await fetch(api.address + '/users', {
      method: 'POST',
      body: JSON.stringify(user),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })

    expect(resp.status).toBe(201)
  })

  test('/users get route', async () => {
    const resp = await fetch(api.address + '/users', {
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data.length).toBeGreaterThanOrEqual(1)

    const foundUser = d.data.find(curUser => curUser.username === user.username)
    expect(foundUser).not.toBe(undefined)

    user = Object.assign(user, foundUser)
  })

  test('/users get route unauthorized', async () => {
    const resp = await fetch(api.address + '/users')
    expect(resp.status).toBe(403)
  })

  test('/users/{id} get route', async () => {
    const resp = await fetch(api.address + '/users/' + user.id, {
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data).toEqual({
      id: user.id,
      username: user.username,
      firstName: user.firstName,
      lastName: user.lastName,
      stars: user.stars,
      roles: user.roles,
    })
  })

  test('/users/{id} get route unauthorized', async () => {
    const resp = await fetch(api.address + '/users/' + user.id)

    expect(resp.status).toBe(403)
  })

  test('/users/{id} complete admin patch route', async () => {
    const patchUser = {
      id: user.id,
      username: user.username + 'foo',
      password: user.password + 'b',
      firstName: user.firstName + 'bar',
      lastName: user.lastName + 'foo',
      stars: (user.stars || []).concat('2018flor'),
      roles: {},
    }
    patchUser.roles.isAdmin = !(user.roles.isAdmin || true)

    const resp = await fetch(api.address + '/users/' + patchUser.id, {
      method: 'PATCH',
      body: JSON.stringify(patchUser),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })

    expect(resp.status).toBe(204)

    user = Object.assign(user, patchUser)
  })

  test('/users/{id} partial admin patch route', async () => {
    const patchUser = {
      username: user.username + 'bar',
      roles: { isVerified: true },
    }

    const resp = await fetch(api.address + '/users/' + user.id, {
      method: 'PATCH',
      body: JSON.stringify(patchUser),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })

    expect(resp.status).toBe(204)

    user = Object.assign(user, patchUser)
  })

  test('/users/{id} get self route', async () => {
    const resp = await fetch(api.address + '/users/' + user.id, {
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT(user)),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data).toEqual({
      id: user.id,
      username: user.username,
      firstName: user.firstName,
      lastName: user.lastName,
      stars: user.stars,
      roles: { ...user.roles, isAdmin: false },
    })
  })

  test('/users/{id} complete self patch route', async () => {
    const patchUser = {
      username: user.username + 'foo',
      password: user.password + 'bla',
      firstName: user.firstName + 'bar',
      lastName: user.lastName + 'foo',
      stars: (user.stars || []).concat('2018flor_qm29'),
    }

    const resp = await fetch(api.address + '/users/' + user.id, {
      method: 'PATCH',
      body: JSON.stringify(user),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT(user)),
      },
    })

    expect(resp.status).toBe(204)

    user = Object.assign(user, patchUser)
  })
})

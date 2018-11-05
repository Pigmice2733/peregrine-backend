const api = require('./../api.test')
const fetch = require('node-fetch')

const realm = {
  team: 'frc1234',
  name: 'Numb',
  publicData: false,
}

let realmAdmin

describe('/realms endpoint', () => {
  test('/realms create unauthorized', async () => {
    const realmResp = await fetch(api.address + '/realms', {
      method: 'POST',
      body: JSON.stringify(realm),
    })
    expect(realmResp.status).toBe(401)
  })

  test('/realms create', async () => {
    const realmResp = await fetch(api.address + '/realms', {
      method: 'POST',
      body: JSON.stringify(realm),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })
    expect(realmResp.status).toBe(200)
    const d = await realmResp.json()
    realmAdmin = d.data

    expect(realmAdmin.roles.isAdmin).toEqual(true)
    expect(realmAdmin.roles.isSuperAdmin).toEqual(false)
    expect(realmAdmin.roles.isVerified).toEqual(true)
    expect(realmAdmin.realm).toEqual(realm.team)
  })

  test('/realms create unathorized', async () => {
    const realmResp = await fetch(api.address + '/realms', {
      method: 'POST',
      body: JSON.stringify(realm),
    })
    expect(realmResp.status).toBe(401)
  })

  test('/realms get super-admin', async () => {
    const resp = await fetch(api.address + '/realms', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data.length).toBeGreaterThanOrEqual(1)
    const foundRealm = d.data.find(curRealm => curRealm.team === realm.team)

    expect(foundRealm).toEqual({
      team: realm.team,
      name: realm.name,
      publicData: realm.publicData,
    })
    expect(Object.keys(foundRealm)).toEqual(['team', 'name', 'publicData'])
  })

  test('/realms get unauthorized', async () => {
    const resp = await fetch(api.address + '/realms', {
      method: 'GET',
    })
    expect(resp.status).toBe(401)
  })
})

describe('/realms/{teamKey} endpoint', () => {
  test('/realms/{teamKey} get super-admin', async () => {
    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data).toEqual({
      team: realm.team,
      name: realm.name,
      publicData: realm.publicData,
    })
    expect(Object.keys(d.data)).toEqual(['team', 'name', 'publicData'])
  })

  test('/realms/{teamKey} get admin', async () => {
    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data).toEqual({
      team: realm.team,
      name: realm.name,
      publicData: realm.publicData,
    })
    expect(Object.keys(d.data)).toEqual(['team', 'name', 'publicData'])
  })

  test('/realms/{teamKey} get unauthorized', async () => {
    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'GET',
    })
    expect(resp.status).toBe(401)
  })

  test('/realms/{teamKey} patch unauthorized', async () => {
    const patchRealm = {
      name: 'Fake',
    }

    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'PATCH',
      body: JSON.stringify(patchRealm),
    })

    expect(resp.status).toBe(401)
  })

  test('/realms/{teamKey} patch non-existent', async () => {
    const patchRealm = {
      team: 'blah',
      name: 'Real',
    }

    const resp = await fetch(
      api.address + '/realms/very_non_existent_and_fake',
      {
        method: 'PATCH',
        body: JSON.stringify(patchRealm),
        headers: {
          'Content-Type': 'application/json',
          Authentication: 'Bearer ' + (await api.getJWT()),
        },
      },
    )
    expect(resp.status).toBe(404)
  })

  test('/realms/{teamKey} complete patch', async () => {
    const patchRealm = {
      name: 'Name',
      publicData: !realm.publicData,
    }

    const resp = await fetch(api.address + '/realms/' + realm.team, {
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
  })

  test('/realms/{teamKey} partial patch', async () => {
    const patchRealm = {
      team: 'blah',
      name: 'Real',
    }

    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'PATCH',
      body: JSON.stringify(patchRealm),
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
      },
    })

    realm.name = 'Real'

    expect(resp.status).toBe(204)
  })

  test('check that patches succeeded', async () => {
    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT(realmAdmin)),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()

    expect(d.data).toEqual({
      team: realm.team,
      name: realm.name,
      publicData: realm.publicData,
    })
    expect(Object.keys(d.data)).toEqual(['team', 'name', 'publicData'])
  })

  test('/realms/{teamKey} delete unauthorized', async () => {
    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'DELETE',
    })

    expect(resp.status).toBe(401)
  })

  test('/realms/{teamKey} delete authorized', async () => {
    const resp = await fetch(api.address + '/realms/' + realm.team, {
      method: 'DELETE',
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })

    expect(resp.status).toBe(204)
  })

  test('test that deletes succeeded', async () => {
    const resp = await fetch(api.address + '/realms', {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
        Authentication: 'Bearer ' + (await api.getJWT()),
      },
    })
    expect(resp.status).toBe(200)

    const d = await resp.json()
    const foundRealm = d.data.find(curRealm => curRealm.team === realm.team)

    expect(foundRealm).toBeUndefined()
  })
})

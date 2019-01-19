const api = require('./../api.test')
const fetch = require('node-fetch')

test('schemas', async () => {
  let resp = await fetch(api.address + '/schemas')
  expect(resp.status).toBe(200)

  let schema = {
    year: 2018,
    auto: [
      {
        name: 'Crossed Line',
        type: 'boolean',
      },
    ],
    teleop: [
      {
        name: 'Fuel',
        type: 'number',
      },
      {
        name: 'Cubes',
        type: 'number',
      },
    ],
  }
  resp = await fetch(api.address + '/schemas', {
    method: 'POST',
    body: JSON.stringify(schema),
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(201)

  resp = await fetch(api.address + '/schemas', {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)

  let d = await resp.json()
  let foundSchema = d.find(curSchema => schema.year === curSchema.year)

  resp = await fetch(api.address + `/schemas/${foundSchema.id}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d.year).toEqual(schema.year)
  expect(d.realmId).toBeUndefined()
  expect(d.id).not.toBeUndefined()
  expect(d.auto).not.toBeUndefined()
  expect(d.teleop).not.toBeUndefined()

  d.auto.forEach(stat => {
    expect(stat.name).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  d.teleop.forEach(stat => {
    expect(stat.name).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  expect(Object.keys(d)).toBeASubsetOf([
    'id',
    'realmId',
    'year',
    'auto',
    'teleop',
  ])

  resp = await fetch(api.address + `/schemas/year/${schema.year}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d.year).toEqual(schema.year)
  expect(d.realmId).toBeUndefined()
  expect(d.id).not.toBeUndefined()
  expect(d.auto).not.toBeUndefined()
  expect(d.teleop).not.toBeUndefined()

  d.auto.forEach(stat => {
    expect(stat.name).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  d.teleop.forEach(stat => {
    expect(stat.name).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  expect(Object.keys(d)).toBeASubsetOf([
    'id',
    'realmId',
    'year',
    'auto',
    'teleop',
  ])
})

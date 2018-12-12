const api = require('./../api.test')
const fetch = require('node-fetch')

test('schemas', async () => {
  let resp = await fetch(api.address + '/schemas')
  expect(resp.status).toBe(200)

  let d = await resp.json()
  expect(d.data).toHaveLength(1)

  let schema = {
    year: 2018,
    auto: [
      {
        statName: 'Crossed Line',
        type: 'boolean',
      },
    ],
    teleop: [
      {
        statName: 'Fuel',
        type: 'number',
      },
      {
        statName: 'Cubes',
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

  d = await resp.json()
  expect(d.data).toHaveLength(2)
  let foundSchema = d.data.find(curSchema => schema.year === curSchema.year)

  resp = await fetch(api.address + `/schemas/${foundSchema.id}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + (await api.getJWT()),
    },
  })
  expect(resp.status).toBe(200)
  d = await resp.json()

  expect(d.data.year).toEqual(schema.year)
  expect(d.data.realmId).toBeUndefined()
  expect(d.data.id).not.toBeUndefined()
  expect(d.data.auto).not.toBeUndefined()
  expect(d.data.teleop).not.toBeUndefined()

  d.data.auto.forEach(stat => {
    expect(stat.statName).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  d.data.teleop.forEach(stat => {
    expect(stat.statName).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  expect(Object.keys(d.data)).toBeASubsetOf([
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

  expect(d.data.year).toEqual(schema.year)
  expect(d.data.realmId).toBeUndefined()
  expect(d.data.id).not.toBeUndefined()
  expect(d.data.auto).not.toBeUndefined()
  expect(d.data.teleop).not.toBeUndefined()

  d.data.auto.forEach(stat => {
    expect(stat.statName).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  d.data.teleop.forEach(stat => {
    expect(stat.statName).not.toBeUndefined()
    expect(stat.type).not.toBeUndefined()
  })

  expect(Object.keys(d.data)).toBeASubsetOf([
    'id',
    'realmId',
    'year',
    'auto',
    'teleop',
  ])
})

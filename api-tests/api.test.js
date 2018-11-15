const fetch = require('node-fetch')
const jsyaml = require('js-yaml')
const fs = require('fs')

expect.extend(require('./matchers.js'))

const config = jsyaml.safeLoad(
  fs.readFileSync(
    `./../etc/config.${process.env.GO_ENV || 'development'}.yaml`,
    'utf8',
  ),
)

const address = `http://${config.server.httpAddress}`

const youtubeOrTwitch = /^(youtube|twitch)$/

const seedUser = {
  username: 'test',
  password: 'testpassword',
}

module.exports = {
  address,
  config,
  youtubeOrTwitch,
  getJWT: async (user = seedUser) => {
    const resp = await fetch(address + '/authenticate', {
      method: 'POST',
      body: JSON.stringify({
        username: user.username,
        password: user.password,
      }),
      headers: { 'Content-Type': 'application/json' },
    })
    const d = await resp.json()
    return d.data.jwt
  },
}

test('the api is listening', () => {
  return fetch(address + '/')
})

test('the api is healthy', async () => {
  const resp = await fetch(address + '/')
  expect(resp.status).toBe(200)

  const d = await resp.json()

  expect(d.data.ok).toBe(true)
  expect(d.data.listen.http).toBe(config.server.httpAddress)
  expect(d.data.services.tba).toBe(true)
  expect(d.data.services.postgresql).toBe(true)
})

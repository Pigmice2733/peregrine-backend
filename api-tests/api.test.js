import fetch from 'node-fetch'
import config from './../etc/config.development.json'

const addr = `http://${config.server.address}/`

test('the api is alive', () => {
  return fetch(addr)
})

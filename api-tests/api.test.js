const fetch = require('node-fetch')
const config = require('./../etc/config.development.json')

const addr = `http://${config.server.address}/`

test('the api is alive', () => {
    return fetch(addr)
})
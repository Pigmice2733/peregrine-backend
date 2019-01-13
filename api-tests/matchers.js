module.exports = {
  toBeAnInt(received) {
    const pass = Number.isInteger(received)
    const message = pass
      ? () => `expected ${received} not to be an integer`
      : () => `expected ${received} to be an integer`
    return {
      message,
      pass,
    }
  },
  toBeADateString(received) {
    const parsedDate = new Date(received)
    const pass = !isNaN(Number(parsedDate))
    const message = pass
      ? () => `expected ${received} to not be a valid date string`
      : () => `expected ${received} to be a valid date string`
    return { pass, message }
  },
  toEqualDate(recieved, expected) {
    const parsedRecievedDate = new Date(recieved)
    const parsedExpectedDate = new Date(expected)
    return {
      pass: Number(parsedRecievedDate) === Number(parsedExpectedDate),
      message: `expected ${recieved} to equal ${expected}`,
    }
  },
  toBeA(received, type) {
    try {
      expect(received).toEqual(expect.any(type))
    } catch (error) {
      return { message: error.matcherResult.message, pass: false }
    }
    return { pass: true }
  },
  toBeATeamKey(received) {
    try {
      expect(received).toMatch(/^frc([1-9a-zA-Z])+/)
    } catch (error) {
      return {
        message: () => `expected ${received} to be a team key`,
        pass: false,
      }
    }
    return { pass: true }
  },
  toBeAnEvent(received) {
    try {
      expect(received.name).toBeA(String)
      expect(received.startDate).toBeADateString()
      expect(received.endDate).toBeADateString()
      expect(received.locationName).toBeA(String)
      expect(received.lat).toBeA(Number)
      expect(received.lon).toBeA(Number)
      expect(received.key).toBeA(String)
      expect(received.district).toBeUndefinedOr(String)
      expect(received.fullDistrict).toBeUndefinedOr(String)
      expect(received.week).toBeUndefinedOr(Number)
      expect(received.webcasts).toBeA(Array)
      expect(Object.keys(received)).toBeASubsetOf([
        'key',
        'realmId',
        'schemaId',
        'name',
        'week',
        'startDate',
        'endDate',
        'locationName',
        'lat',
        'lon',
        'district',
        'fullDistrict',
        'webcasts',
      ])
    } catch (error) {
      return {
        message: () => `expected to get an event. failed:\n ${error}`,
        pass: false,
      }
    }
    return { pass: true }
  },
  toBeAMatch(received) {
    try {
      expect(received.key).toBeA(String)
      expect(received.time).toBeADateString()
      expect(received.redScore).toBeUndefinedOr(Number)
      expect(received.blueScore).toBeUndefinedOr(Number)
      expect(received.redAlliance).toEqual(expect.any(Array))
      expect(received.redAlliance).toHaveLength(3)
      received.redAlliance.forEach(team => {
        expect(team).toBeATeamKey()
      })
      expect(received.blueAlliance).toEqual(expect.any(Array))
      expect(received.blueAlliance).toHaveLength(3)
      received.blueAlliance.forEach(team => {
        expect(team).toBeATeamKey()
      })
      expect(Object.keys(received)).toBeASubsetOf([
        'key',
        'time',
        'scheduledTime',
        'redAlliance',
        'blueAlliance',
        'redScore',
        'blueScore',
      ])
    } catch (error) {
      return {
        message: () => `expected to get a match. failed:\n ${error}`,
        pass: false,
      }
    }
    return { pass: true }
  },
  toIncludeTeam(received, team) {
    if (
      received.blueAlliance.includes(team) ||
      received.redAlliance.includes(team)
    ) {
      return { pass: true }
    }
    return {
      message: () => `expected match ${received} to include team ${team}`,
      pass: false,
    }
  },
  toBeUndefinedOr(received, type) {
    if (received === undefined) {
      return { pass: true }
    }
    try {
      expect(received).toEqual(expect.any(type))
    } catch (error) {
      return { message: error.matcherResult.message, pass: false }
    }
    return { pass: true }
  },
  toBeASubsetOf(received, items) {
    const s = new Set(items)
    let unexpected = received.reduce(
      (unexpected, i) => (s.has(i) ? unexpected : unexpected.concat(i)),
      [],
    )
    const pass = unexpected.length === 0
    const message = pass
      ? () => `did not expect item(s): ${unexpected}`
      : () => `did not expect item(s): ${unexpected}`
    return { message, pass }
  },
}

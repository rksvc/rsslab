Date = class extends Date {
  constructor(...args) {
    if (args.length === 1 && typeof args[0] === 'string') {
      const m = args[0].match(
        /^(\d{4})\/(\d{1,2})\/(\d{1,2}) (\d{1,2}):(\d{1,2}):(\d{1,2}) GMT\+8$/,
      )
      if (m) {
        // 2006-01-02T15:04:05Z0700
        const p = s => s.padStart(2, '0')
        args[0] = `${m[1]}-${p(m[2])}-${p(m[3])}T${p(m[4])}:${p(m[5])}:${p(m[6])}+0800`
      }
    }
    super(...args)
  }
}

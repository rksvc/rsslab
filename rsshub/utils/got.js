;(exports, require, module) => {
  function parseArgs(request, options = {}) {
    if (typeof request === 'string' || request instanceof URL) {
      options.url = request.toString()
      return options
    }
    return request
  }

  function got(request, options) {
    const opts = parseArgs(request, options)
    if (opts.prefixUrl) {
      opts.baseURL = opts.prefixUrl
      delete opts.prefixUrl
    }
    if (opts.searchParams) {
      opts.query = opts.searchParams
      delete opts.searchParams
    }
    return $fetch(opts)
  }

  for (const method of ['get', 'post', 'put', 'head', 'patch', 'delete']) {
    got[method] = (request, options) =>
      got({ ...parseArgs(request, options), method: method.toUpperCase() })
  }

  module.exports = got
}

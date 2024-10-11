;(exports, require, module) => {
  function raw(request, options = {}) {
    if (typeof request === 'string' || request instanceof URL) {
      options.url = request.toString()
      return $fetch(options)
    }
    return $fetch(request)
  }

  async function ofetch(request, options) {
    const response = await raw(request, options)
    return response.data
  }

  ofetch.raw = raw

  module.exports = ofetch
}

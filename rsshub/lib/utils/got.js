function parseArgs(request, options = {}) {
  if (typeof request === 'string' || request instanceof URL) {
    options.url = request.toString();
  } else {
    options = request;
  }
  return options;
}

function got(request, options) {
  options = parseArgs(request, options);
  if (options.prefixUrl) {
    options.baseURL = options.prefixUrl;
    delete options.prefixUrl;
  }
  if (options.searchParams) {
    options.query = options.searchParams;
    delete options.searchParams;
  }
  return $fetch(options);
}

got.get = (request, options) => got({ ...parseArgs(request, options), method: 'GET' });
got.post = (request, options) => got({ ...parseArgs(request, options), method: 'POST' });
got.put = (request, options) => got({ ...parseArgs(request, options), method: 'PUT' });
got.head = (request, options) => got({ ...parseArgs(request, options), method: 'HEAD' });
got.patch = (request, options) =>
  got({ ...parseArgs(request, options), method: 'PATCH' });
got.delete = (request, options) =>
  got({ ...parseArgs(request, options), method: 'DELETE' });

module.exports = got;

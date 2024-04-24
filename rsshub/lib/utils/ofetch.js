function parseArgs(request, options = {}) {
  if (typeof request === 'string' || request instanceof URL) {
    options.url = request.toString();
  } else {
    options = request;
  }
  return options;
}

function raw(request, options) {
  return $fetch(parseArgs(request, options));
}

async function ofetch(request, options) {
  const response = await raw(request, options);
  return response.data;
}

ofetch.raw = (request, options) => raw(request, options);

module.exports = ofetch;

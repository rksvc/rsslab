"use strict";

(function() {
  var xfetch = function(resource, init) {
    init = init || {}
    if (['post', 'put', 'delete'].indexOf(init.method) !== -1) {
      init['headers'] = init['headers'] || {}
      init['headers']['x-requested-by'] = 'RSSLab'
    }
    return fetch(resource, init)
  }
  var api = function(method, endpoint, data) {
    var headers = {'Content-Type': 'application/json'}
    return xfetch(endpoint, {
      method: method,
      headers: headers,
      body: JSON.stringify(data),
    })
  }

  var json = function(res) {
    return res.json().then(function (json) {
      if (json.error) {
        alert(json.error)
        throw new Error(json.error)
      }
      return json
    })
  }

  var check = function(res) {
    return res.ok || json(res)
  }

  var param = function(query) {
    if (!query) return ''
    return '?' + Object.keys(query).map(function(key) {
      return encodeURIComponent(key) + '=' + encodeURIComponent(query[key])
    }).join('&')
  }

  window.api = {
    feeds: {
      list: function() {
        return api('get', './api/feeds').then(json)
      },
      create: function(data) {
        return api('post', './api/feeds', data).then(json)
      },
      update: function(id, data) {
        return api('put', './api/feeds/' + id, data).then(check)
      },
      delete: function(id) {
        return api('delete', './api/feeds/' + id).then(check)
      },
      list_items: function(id) {
        return api('get', './api/feeds/' + id + '/items').then(json)
      },
      refresh: function(id) {
        return api('post', id ? './api/feeds/' + id + '/refresh' : './api/feeds/refresh').then(check)
      },
      refresh_errors: function() {
        return api('post', './api/feeds/errors/refresh').then(check)
      },
      list_errors: function() {
        return api('get', './api/feeds/errors').then(json)
      },
    },
    folders: {
      list: function() {
        return api('get', './api/folders').then(json)
      },
      create: function(data) {
        return api('post', './api/folders', data).then(json)
      },
      update: function(id, data) {
        return api('put', './api/folders/' + id, data).then(check)
      },
      delete: function(id) {
        return api('delete', './api/folders/' + id).then(check)
      },
      refresh: function(id) {
        return api('post', './api/folders/' + id + '/refresh').then(check)
      },
    },
    items: {
      get: function(id) {
        return api('get', './api/items/' + id).then(json)
      },
      list: function(query) {
        return api('get', './api/items' + param(query)).then(json)
      },
      update: function(id, data) {
        return api('put', './api/items/' + id, data).then(check)
      },
      mark_read: function(query) {
        return api('put', './api/items' + param(query)).then(check)
      },
    },
    settings: {
      get: function() {
        return api('get', './api/settings').then(json)
      },
      update: function(data) {
        return api('put', './api/settings', data).then(check)
      },
    },
    status: function() {
      return api('get', './api/status').then(json)
    },
    upload_opml: function(form) {
      return xfetch('./api/opml/import', {
        method: 'post',
        body: new FormData(form),
      }).then(check)
    }
  }
})()

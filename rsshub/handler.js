;async (sourcePath, ctx) => {
  const data = await require(`./${sourcePath}`).route.handler(ctx)
  for (const item of data.item ?? []) {
    for (const key of ['pubDate', 'updated']) {
      if (item[key]) {
        const date = new Date(item[key])
        if (Number.isNaN(date.valueOf())) {
          delete item[key]
        } else {
          item[key] = date.toISOString()
        }
      }
    }
  }
  return data
}

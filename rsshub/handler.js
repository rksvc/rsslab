;async (sourcePath, ctx) => {
  const data = await require(`./${sourcePath}`).route.handler(ctx)
  for (const item of data.item ?? [])
    for (const key of ['pubDate', 'updated'])
      if (item[key]) item[key] = new Date(item[key]).toISOString()
  return data
}

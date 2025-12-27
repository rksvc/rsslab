import { useEffect, useState } from 'react'

export default function RelativeTime({ date, format }: { date: string; format: (date: string) => string }) {
  const [formatted, setFormatted] = useState(format(date))

  useEffect(() => {
    const interval = setInterval(() => {
      setFormatted(format(date))
    }, 60_000)
    return () => clearInterval(interval)
  }, [date, format])

  return formatted
}

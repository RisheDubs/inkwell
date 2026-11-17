import { Link } from 'react-router'

export default function NotFound() {
  return (
    <div>
      <h1 className="text-2xl font-semibold">Page not found</h1>
      <Link to="/" className="mt-2 inline-block text-red-600 underline">Back to your journal</Link>
    </div>
  )
}
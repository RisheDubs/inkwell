import { Link } from 'react-router'

// The one answer for an unknown handle, a private journal, a missing entry or any error.
// Same words every time, like the API's single 404, so the page never hints that a private
// journal exists.
export default function PublicNotFound() {
  return (
    <div>
      <h1 className="text-2xl font-semibold">This journal isn’t available</h1>
      <p className="mt-2 text-sm text-stone-600">
        The link may be wrong, or the journal isn’t shared publicly.
      </p>
      <Link to="/" className="mt-2 inline-block text-red-600 underline">
        Go to Inkwell
      </Link>
    </div>
  )
}
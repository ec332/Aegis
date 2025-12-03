export default function LoadingSkeleton({ lines = 3 }: { lines?: number }) {
  return (
    <div role="status" aria-live="polite" className="animate-pulse space-y-3">
      {Array.from({ length: lines }).map((_, i) => (
        <div key={i} className="h-4 bg-gray-200 rounded" />
      ))}
    </div>
  );
}

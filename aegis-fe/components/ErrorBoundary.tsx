'use client';

import { useEffect } from 'react';
import { useAppStore } from '@/store/appStore';

export default function ErrorBoundary({
  children,
}: {
  children: React.ReactNode;
}) {
  const { error, clearError, isBackendHealthy, checkBackendHealth } = useAppStore();

  useEffect(() => {
    // Check backend health on mount and periodically
    checkBackendHealth();
    const interval = setInterval(checkBackendHealth, 30000); // Check every 30 seconds
    return () => clearInterval(interval);
  }, [checkBackendHealth]);

  useEffect(() => {
    // Auto-clear errors after 5 seconds
    if (error) {
      const timer = setTimeout(clearError, 5000);
      return () => clearTimeout(timer);
    }
  }, [error, clearError]);

  if (error) {
    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg p-6 max-w-md mx-4">
          <div className="flex items-center mb-4">
            <div className="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center mr-3">
              <svg className="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-gray-900">API Error</h3>
          </div>
          <p className="text-gray-600 mb-2">
            <strong>Status:</strong> {error.status}
          </p>
          <p className="text-gray-600 mb-4">
            <strong>Message:</strong> {error.message}
          </p>
          {error.details && (
            <pre className="bg-gray-100 p-2 rounded text-sm text-gray-700 mb-4 overflow-auto">
              {JSON.stringify(error.details, null, 2)}
            </pre>
          )}
          <button
            onClick={clearError}
            className="w-full bg-blue-600 text-white py-2 px-4 rounded hover:bg-blue-700 transition-colors"
          >
            Dismiss
          </button>
        </div>
      </div>
    );
  }

  return (
    <>
      {!isBackendHealthy && (
        <div className="fixed top-4 right-4 bg-yellow-100 border border-yellow-400 text-yellow-800 px-4 py-3 rounded-lg shadow-lg z-40">
          <div className="flex items-center">
            <svg className="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
            </svg>
            <span>Backend connection issues detected. Some features may be limited.</span>
          </div>
        </div>
      )}
      {children}
    </>
  );
}
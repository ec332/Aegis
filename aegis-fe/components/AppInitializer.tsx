'use client';

import { useEffect, useRef } from 'react';
import { useAppStore } from '@/store/appStore';

export default function AppInitializer() {
  const { initializeApp } = useAppStore();
  const ranRef = useRef(false);

  useEffect(() => {
    if (ranRef.current) return;
    ranRef.current = true;
    initializeApp();
  }, [initializeApp]);

  return null;
}

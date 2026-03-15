import { useEffect, useRef } from "react";
import { QRCode, SVGRenderer as SVG } from "@forward-software/qrcodets";

interface QRCodeDisplayProps {
  value: string;
  size?: number;
  className?: string;
}

export function QRCodeDisplay({ value, size = 128, className = "" }: QRCodeDisplayProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const el = containerRef.current;
    if (!el || !value) return;

    el.innerHTML = "";
    new QRCode(value, {
      size,
      correctionLevel: "M",
      colorDark: "#000000",
      colorLight: "#ffffff",
    }).renderTo(SVG(el));
  }, [value, size]);

  return <div ref={containerRef} className={className} role="img" aria-label="QR code" />;
}

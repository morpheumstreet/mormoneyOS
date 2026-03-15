import { QRCodeDisplay } from "./QRCodeDisplay";

interface QRModalProps {
  address: string;
  label: string;
  onClose: () => void;
}

export function QRModal({ address, label, onClose }: QRModalProps) {
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="electric-card p-6 max-w-sm"
        onClick={(e) => e.stopPropagation()}
      >
        <p className="text-sm font-medium text-[#9bb7eb] mb-2">{label}</p>
        <div className="bg-white p-4 rounded-lg inline-block">
          <QRCodeDisplay value={address} size={128} />
        </div>
        <p className="mt-2 font-mono text-xs text-[#8aa8df] break-all">{address}</p>
        <button
          type="button"
          onClick={onClose}
          className="mt-4 w-full electric-button py-2 rounded-lg text-sm"
        >
          Close
        </button>
      </div>
    </div>
  );
}

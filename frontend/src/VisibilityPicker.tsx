import { CheckCircle2, Globe2, LockKeyhole } from 'lucide-react';

export function VisibilityPicker({
  value,
  onChange,
  disabled = false,
}: {
  value: 'public' | 'private';
  onChange: (value: 'public' | 'private') => void;
  disabled?: boolean;
}) {
  return (
    <fieldset className="visibility-picker" disabled={disabled}>
      <legend>دسترسی اتاق</legend>
      {(
        [
          ['private', 'خصوصی', 'فقط با لینک دعوت', LockKeyhole],
          ['public', 'عمومی', 'نمایش در صفحهٔ اصلی برای همه', Globe2],
        ] as const
      ).map(([id, title, description, Icon]) => (
        <label key={id} className={`visibility-option ${value === id ? 'selected' : ''}`}>
          <input
            type="radio"
            name="room-visibility"
            value={id}
            checked={value === id}
            onChange={() => onChange(id)}
          />
          <Icon size={20} />
          <span>
            <strong>{title}</strong>
            <small>{description}</small>
          </span>
          {value === id && <CheckCircle2 className="visibility-check" size={18} />}
        </label>
      ))}
    </fieldset>
  );
}

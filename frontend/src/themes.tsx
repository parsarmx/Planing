import { Check, Palette } from 'lucide-react';

export const ROOM_THEMES = [
  { id: 'violet', name: 'بنفش', description: 'آرام و آشنا' },
  { id: 'pink', name: 'صورتی', description: 'شاد و صمیمی' },
  { id: 'blue', name: 'آبی', description: 'شفاف و متمرکز' },
  { id: 'mint', name: 'سبز نعنایی', description: 'تازه و آرام' },
  { id: 'amber', name: 'کهربایی', description: 'گرم و پرانرژی' },
  { id: 'coral', name: 'مرجانی', description: 'خلاق و سرزنده' },
] as const;
export type RoomTheme = (typeof ROOM_THEMES)[number]['id'];
export function roomTheme(value?: string): RoomTheme {
  return ROOM_THEMES.find((t) => t.id === value)?.id ?? 'violet';
}
export function ThemePicker({
  value,
  onChange,
  disabled = false,
  readOnly = false,
}: {
  value: RoomTheme;
  onChange: (theme: RoomTheme) => void;
  disabled?: boolean;
  readOnly?: boolean;
}) {
  return (
    <div className="room-theme-picker">
      <div className="theme-picker-heading">
        <span>
          <Palette size={16} />
          رنگ تم اتاق
        </span>
        <small>{readOnly ? 'انتخاب سازندهٔ اتاق' : 'یک رنگ برای میزِ تیم شما'}</small>
      </div>
      <div className="room-theme-options" role="group" aria-label="تم‌های رنگی اتاق">
        {ROOM_THEMES.map((theme) => (
          <button
            key={theme.id}
            type="button"
            className={`room-theme-option ${value === theme.id ? 'selected' : ''}`}
            data-palette={theme.id}
            aria-label={`تم ${theme.name}`}
            aria-pressed={value === theme.id}
            disabled={disabled || readOnly}
            onClick={() => onChange(theme.id)}
          >
            <span className="theme-mini-table">
              <i />
              <i />
              <i />
              <b>
                <Palette size={14} />
              </b>
            </span>
            <span className="theme-option-name">
              {theme.name}
              {value === theme.id && <Check size={14} />}
            </span>
            <small>{theme.description}</small>
          </button>
        ))}
      </div>
    </div>
  );
}

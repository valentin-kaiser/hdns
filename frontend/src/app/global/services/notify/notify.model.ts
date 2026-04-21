export type NotificationKind = 'info' | 'error';

export interface NotificationItem {
  id: string;
  kind: NotificationKind;
  message: string;
  title?: string;
  /** Full raw error string shown in the detail modal */
  rawError?: string;
}

export interface WarningButton {
  text: string;
  color?: 'primary' | 'accent' | 'warn' | string;
  handler?: () => void;
}

export interface Warning {
  title: string;
  message: string;
  buttons: WarningButton[];
  showCheckbox?: boolean;
  checkboxLabel?: string;
  checkboxValue?: boolean;
  checkboxChange?: (checked: boolean) => void;
}

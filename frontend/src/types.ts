import type { RoomTheme } from './themes';
import type { AvatarConfig } from './avatars';
export type User = {
  id: string;
  name: string;
  phone: string;
  avatar: string;
  avatar_config?: AvatarConfig;
  role: string;
};
export type Participant = {
  id: string;
  name: string;
  avatar: string;
  avatar_config?: AvatarConfig;
  role: string;
  observer: boolean;
  online: boolean;
  has_voted: boolean;
  vote?: string;
};
export type Round = {
  id: string;
  title: string;
  description: string;
  status: 'pending' | 'voting' | 'revealed' | 'completed';
  estimate: string;
  started_at: string;
};
export type Room = {
  id: string;
  name: string;
  host_id: string;
  deck: string;
  theme: RoomTheme;
  visibility: 'public' | 'private';
  last_activity_at: string;
  cards: string[];
  members: Participant[];
  rounds: Round[];
  active_round_id: string;
  created_at: string;
  closed: boolean;
  revision: number;
  my_vote: string;
  average: number;
  consensus: boolean;
  distribution: Record<string, number>;
};
export type PublicRoom = {
  id: string;
  name: string;
  deck: string;
  theme: RoomTheme;
  host_name: string;
  member_count: number;
  capacity: number;
  created_at: string;
};
export type Command = {
  action: string;
  value?: string;
  round_id?: string;
  title?: string;
  description?: string;
};

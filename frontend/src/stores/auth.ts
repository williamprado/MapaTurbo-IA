import { create } from 'zustand';

interface User {
  id: string;
  email: string;
  name: string;
  global_role: string;
  status: string;
}

interface Organization {
  organization_id: string;
  organization_name: string;
  organization_slug: string;
  organization_status: string;
  role: string;
}

interface AuthState {
  token: string | null;
  user: User | null;
  organizations: Organization[];
  activeOrgId: string | null;
  setAuth: (token: string, user: User, organizations: Organization[]) => void;
  setActiveOrgId: (orgId: string | null) => void;
  logout: () => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  token: localStorage.getItem('token'),
  user: localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!) : null,
  organizations: localStorage.getItem('organizations') ? JSON.parse(localStorage.getItem('organizations')!) : [],
  activeOrgId: localStorage.getItem('activeOrgId'),

  setAuth: (token, user, organizations) => {
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
    localStorage.setItem('organizations', JSON.stringify(organizations));
    
    const activeOrgId = organizations.length > 0 ? organizations[0].organization_id : null;
    if (activeOrgId) {
      localStorage.setItem('activeOrgId', activeOrgId);
    } else {
      localStorage.removeItem('activeOrgId');
    }

    set({ token, user, organizations, activeOrgId });
  },

  setActiveOrgId: (activeOrgId) => {
    if (activeOrgId) {
      localStorage.setItem('activeOrgId', activeOrgId);
    } else {
      localStorage.removeItem('activeOrgId');
    }
    set({ activeOrgId });
  },

  logout: () => {
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    localStorage.removeItem('organizations');
    localStorage.removeItem('activeOrgId');
    set({ token: null, user: null, organizations: [], activeOrgId: null });
  },
}));

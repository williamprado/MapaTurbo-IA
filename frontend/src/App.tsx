import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import type { ReactNode } from 'react';
import { useAuthStore } from './stores/auth';
// Public pages
import Home from './pages/public/Home';
import Prices from './pages/public/Prices';

// Auth pages
import Login from './pages/auth/Login';
import Register from './pages/auth/Register';

// App pages
import Dashboard from './pages/app/Dashboard';
import Billing from './pages/app/Billing';
import Settings from './pages/app/Settings';

// Admin pages
import AdminDashboard from './pages/admin/AdminDashboard';
import ManageOrganizations from './pages/admin/ManageOrganizations';
import ManagePlans from './pages/admin/ManagePlans';
import ManageUsers from './pages/admin/ManageUsers';
import ManageSettings from './pages/admin/ManageSettings';
import ManageSubscriptions from './pages/admin/ManageSubscriptions';
import ViewAuditLogs from './pages/admin/ViewAuditLogs';
import ManageAiProviders from './pages/admin/ManageAiProviders';
import ManagePayments from './pages/admin/ManagePayments';

// Layouts
import AppLayout from './layouts/AppLayout';
import AdminLayout from './layouts/AdminLayout';

function ProtectedRoute({ children, requireAdmin = false }: { children: ReactNode; requireAdmin?: boolean }) {
  const { token, user } = useAuthStore();

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  if (requireAdmin && user?.global_role !== 'SUPER_ADMIN') {
    return <Navigate to="/app" replace />;
  }

  return children;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        {/* Public Routes */}
        <Route path="/" element={<Home />} />
        <Route path="/precos" element={<Prices />} />
        <Route path="/login" element={<Login />} />
        <Route path="/cadastro" element={<Register />} />

        {/* User Workspace (Tenant isolated app) */}
        <Route
          path="/app"
          element={
            <ProtectedRoute>
              <AppLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<Dashboard />} />
          <Route path="maps" element={<Dashboard />} />
          <Route path="billing" element={<Billing />} />
          <Route path="settings" element={<Settings />} />
        </Route>

        {/* Super Admin Dashboard */}
        <Route
          path="/admin"
          element={
            <ProtectedRoute requireAdmin>
              <AdminLayout />
            </ProtectedRoute>
          }
        >
          <Route index element={<AdminDashboard />} />
          <Route path="organizations" element={<ManageOrganizations />} />
          <Route path="plans" element={<ManagePlans />} />
          <Route path="users" element={<ManageUsers />} />
          <Route path="settings" element={<ManageSettings />} />
          <Route path="subscriptions" element={<ManageSubscriptions />} />
          <Route path="audit-logs" element={<ViewAuditLogs />} />
          <Route path="ai-providers" element={<ManageAiProviders />} />
          <Route path="payments" element={<ManagePayments />} />
        </Route>

        {/* Fallback */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  );
}

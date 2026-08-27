import { BrowserRouter, Routes, Route } from "react-router-dom";
import { AuthProvider } from "./lib/auth";
import RequireAuth from "./components/RequireAuth";
import Layout from "./components/Layout";
import Login from "./pages/Login";
import Signup from "./pages/Signup";
import Links from "./pages/Links";
import Analytics from "./pages/Analytics";
import Billing from "./pages/Billing";

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/signup" element={<Signup />} />
          <Route
            element={
              <RequireAuth>
                <Layout />
              </RequireAuth>
            }
          >
            <Route path="/" element={<Links />} />
            <Route path="/analytics" element={<Analytics />} />
            <Route path="/billing" element={<Billing />} />
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link, useLocation } from 'react-router-dom';
import {
  Layout,
  Typography,
  Menu,
} from 'antd';
import {
  DollarCircleOutlined,
  SettingOutlined,
  HomeOutlined,
  InboxOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import Dashboard from './Dashboard';
import Settings from './Settings';
import Archives from './Archives';

const { Header, Content } = Layout;
const { Title } = Typography;

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

const NavMenu: React.FC = () => {
  const location = useLocation();

  const items = [
    {
      key: '/',
      icon: <HomeOutlined />,
      label: <Link to="/">Dashboard</Link>,
    },
    {
      key: '/archives',
      icon: <InboxOutlined />,
      label: <Link to="/archives">Archives</Link>,
    },
    {
      key: '/settings',
      icon: <SettingOutlined />,
      label: <Link to="/settings">Settings</Link>,
    },
    {
      key: 'api-docs',
      icon: <FileTextOutlined />,
      label: (
        <a
          href={`${API_URL}/swagger/index.html`}
          target="_blank"
          rel="noopener noreferrer"
          style={{ color: 'inherit' }}
        >
          API Docs
        </a>
      ),
    },
  ];

  return (
    <Menu
      theme="dark"
      mode="horizontal"
      selectedKeys={[location.pathname]}
      items={items}
      style={{ flex: 1, minWidth: 0 }}
    />
  );
};

function App() {
  return (
    <Router>
      <Layout style={{ minHeight: '100vh' }}>
        <Header
          style={{
            background: '#001529',
            display: 'flex',
            alignItems: 'center',
            padding: '0 24px'
          }}
        >
          <Title level={2} style={{ color: 'white', margin: 0, marginRight: '24px' }}>
            <DollarCircleOutlined style={{ marginRight: 8 }} />
            Joint Analysis
          </Title>
          <NavMenu />
        </Header>

        <Content>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/archives" element={<Archives />} />
            <Route path="/settings" element={<Settings />} />
          </Routes>
        </Content>
      </Layout>
    </Router>
  );
}

export default App;
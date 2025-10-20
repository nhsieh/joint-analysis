import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Layout,
  Typography,
  Card,
  Table,
  Button,
  Upload,
  Input,
  Select,
  Space,
  Row,
  Col,
  Statistic,
  message,
  Spin,
} from 'antd';
import {
  UploadOutlined,
  UserAddOutlined,
  DollarCircleOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import { UploadProps, RcFile } from 'antd/es/upload';
import { ColumnsType } from 'antd/es/table';

interface Transaction {
  id: number;
  description: string;
  amount: number;
  assigned_to: string;
  date_uploaded: string;
  file_name: string;
}

interface Person {
  id: number;
  name: string;
}

interface PersonTotal {
  name: string;
  total: number;
}

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { Option } = Select;

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

function App() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [totals, setTotals] = useState<PersonTotal[]>([]);
  const [newPersonName, setNewPersonName] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    fetchTransactions();
    fetchPeople();
    fetchTotals();
  }, []);

  const fetchTransactions = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/transactions`);
      setTransactions(response.data || []);
    } catch (error) {
      console.error('Error fetching transactions:', error);
    }
  };

  const fetchPeople = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/people`);
      setPeople(response.data || []);
    } catch (error) {
      console.error('Error fetching people:', error);
    }
  };

  const fetchTotals = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/totals`);
      setTotals(response.data || []);
    } catch (error) {
      console.error('Error fetching totals:', error);
    }
  };

  const handleFileUpload = async (file: RcFile): Promise<boolean> => {
    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      await axios.post(`${API_URL}/api/upload-csv`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });
      message.success('CSV uploaded successfully!');
      await fetchTransactions();
      await fetchTotals();
    } catch (error) {
      console.error('Error uploading file:', error);
      message.error('Error uploading file. Please try again.');
    } finally {
      setUploading(false);
    }
    return false; // Prevent default upload behavior
  };

  const assignTransaction = async (transactionId: number, personName: string) => {
    try {
      await axios.put(`${API_URL}/api/transactions/${transactionId}/assign`, {
        assigned_to: personName,
      });
      message.success('Transaction assigned successfully!');
      fetchTransactions();
      fetchTotals();
    } catch (error) {
      console.error('Error assigning transaction:', error);
      message.error('Error assigning transaction');
    }
  };

  const createPerson = async () => {
    if (!newPersonName.trim()) {
      message.warning('Please enter a person name');
      return;
    }

    try {
      await axios.post(`${API_URL}/api/people`, { name: newPersonName });
      setNewPersonName('');
      message.success('Person added successfully!');
      fetchPeople();
    } catch (error) {
      console.error('Error creating person:', error);
      message.error('Error creating person');
    }
  };

  const uploadProps: UploadProps = {
    name: 'file',
    accept: '.csv',
    beforeUpload: handleFileUpload,
    showUploadList: false,
    multiple: false,
  };

  const columns: ColumnsType<Transaction> = [
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: '40%',
    },
    {
      title: 'Amount',
      dataIndex: 'amount',
      key: 'amount',
      render: (amount: number) => (
        <Text strong style={{ color: amount >= 0 ? '#52c41a' : '#ff4d4f' }}>
          ${amount.toFixed(2)}
        </Text>
      ),
      sorter: (a: Transaction, b: Transaction) => a.amount - b.amount,
      width: '15%',
    },
    {
      title: 'Assigned To',
      dataIndex: 'assigned_to',
      key: 'assigned_to',
      render: (assignedTo: string) =>
        assignedTo ? (
          <Text>{assignedTo}</Text>
        ) : (
          <Text type="secondary" italic>Unassigned</Text>
        ),
      width: '20%',
    },
    {
      title: 'Action',
      key: 'action',
      render: (_: any, record: Transaction) => (
        <Select
          style={{ width: 150 }}
          placeholder="Select Person"
          value={record.assigned_to || undefined}
          onChange={(value: string) => assignTransaction(record.id, value)}
          allowClear
        >
          {people.map((person) => (
            <Option key={person.id} value={person.name}>
              {person.name}
            </Option>
          ))}
        </Select>
      ),
      width: '25%',
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header
        style={{
          background: '#001529',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '0 24px'
        }}
      >
        <Title level={2} style={{ color: 'white', margin: 0 }}>
          <DollarCircleOutlined style={{ marginRight: 8 }} />
          Joint Analysis - Expense Tracker
        </Title>
      </Header>

      <Content style={{ padding: '24px', background: '#f0f2f5' }}>
        <div style={{ maxWidth: 1200, margin: '0 auto' }}>
          {/* Upload and Add Person Section */}
          <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
            <Col xs={24} md={12}>
              <Card
                title={
                  <span>
                    <UploadOutlined style={{ marginRight: 8 }} />
                    Upload CSV File
                  </span>
                }
                variant='borderless'
                hoverable
              >
                <Upload {...uploadProps}>
                  <Button
                    icon={<UploadOutlined />}
                    loading={uploading}
                    size="large"
                    type="dashed"
                    style={{ width: '100%', height: 60 }}
                  >
                    <div>
                      <div>{uploading ? 'Uploading...' : 'Click to Upload CSV'}</div>
                      <Text type="secondary" style={{ fontSize: 12 }}>
                        Select a CSV file with expense data
                      </Text>
                    </div>
                  </Button>
                </Upload>
              </Card>
            </Col>

            <Col xs={24} md={12}>
              <Card
                title={
                  <span>
                    <UserAddOutlined style={{ marginRight: 8 }} />
                    Add Person
                  </span>
                }
                variant='borderless'
                hoverable
              >
                <Space.Compact style={{ width: '100%' }}>
                  <Input
                    placeholder="Enter person name"
                    value={newPersonName}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewPersonName(e.target.value)}
                    onPressEnter={createPerson}
                    size="large"
                  />
                  <Button
                    type="primary"
                    onClick={createPerson}
                    icon={<UserAddOutlined />}
                    size="large"
                  >
                    Add
                  </Button>
                </Space.Compact>
              </Card>
            </Col>
          </Row>

          {/* Totals Section */}
          <Card
            title={
              <span>
                <DollarCircleOutlined style={{ marginRight: 8 }} />
                Totals by Person
              </span>
            }
            style={{ marginBottom: 24 }}
            variant='borderless'
            hoverable
          >
            {totals.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '40px 0' }}>
                <Text type="secondary">
                  No data available. Upload a CSV file and assign transactions to see totals.
                </Text>
              </div>
            ) : (
              <Row gutter={[16, 16]}>
                {totals.map((total) => (
                  <Col xs={12} sm={8} md={6} lg={4} key={total.name}>
                    <Card size="small" style={{ textAlign: 'center' }} hoverable>
                      <Statistic
                        title={total.name}
                        value={total.total}
                        precision={2}
                        prefix="$"
                        valueStyle={{
                          color: total.total > 0 ? '#3f8600' : total.total < 0 ? '#cf1322' : '#666666'
                        }}
                      />
                    </Card>
                  </Col>
                ))}
              </Row>
            )}
          </Card>

          {/* Transactions Section */}
          <Card
            title={
              <span>
                <FileTextOutlined style={{ marginRight: 8 }} />
                Transactions ({transactions.length})
              </span>
            }
            variant='borderless'
            hoverable
          >
            <Spin spinning={loading}>
              <Table
                columns={columns}
                dataSource={transactions}
                rowKey="id"
                pagination={{
                  pageSize: 10,
                  showSizeChanger: true,
                  showQuickJumper: true,
                  showTotal: (total: number, range: [number, number]) =>
                    `${range[0]}-${range[1]} of ${total} transactions`,
                  pageSizeOptions: ['10', '20', '50', '100'],
                }}
                scroll={{ x: 800 }}
                locale={{
                  emptyText: (
                    <div style={{ padding: '40px 0' }}>
                      <Text type="secondary">
                        No transactions found. Upload a CSV file to get started.
                      </Text>
                    </div>
                  )
                }}
                size="middle"
              />
            </Spin>
          </Card>
        </div>
      </Content>
    </Layout>
  );
}

export default App;
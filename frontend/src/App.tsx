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
  DeleteOutlined,
} from '@ant-design/icons';
import { UploadProps, RcFile } from 'antd/es/upload';
import { ColumnsType } from 'antd/es/table';

interface Transaction {
  id: string;
  description: string;
  amount: number;
  assigned_to: string;
  date_uploaded: string;
  file_name: string;
  transaction_date: string;
  posted_date: string;
  card_number: string;
  category_id: string;
}

interface Person {
  id: string;
  name: string;
}

interface Category {
  id: string;
  name: string;
  description?: string;
  color?: string;
}

interface PersonTotal {
  person: string;
  total: number;
}

const { Header, Content } = Layout;
const { Title, Text } = Typography;
const { Option } = Select;

const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

function App() {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [totals, setTotals] = useState<PersonTotal[]>([]);
  const [newPersonName, setNewPersonName] = useState('');
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);

  useEffect(() => {
    fetchTransactions();
    fetchPeople();
    fetchCategories();
    fetchTotals();
  }, []);

  const fetchTransactions = async () => {
    try {
      setLoading(true);
      const response = await axios.get(`${API_URL}/api/transactions`);
      setTransactions(response.data || []);
    } catch (error) {
      console.error('Error fetching transactions:', error);
    } finally {
      setLoading(false);
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

  const fetchCategories = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/categories`);
      setCategories(response.data || []);
    } catch (error) {
      console.error('Error fetching categories:', error);
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

  const assignTransaction = async (transactionId: string, personName: string) => {
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

  const deletePerson = async (personId: string, personName: string) => {
    try {
      await axios.delete(`${API_URL}/api/people/${personId}`);
      message.success(`${personName} deleted successfully!`);
      fetchPeople();
      fetchTotals();
    } catch (error) {
      console.error('Error deleting person:', error);
      message.error('Error deleting person');
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
      title: 'Transaction Date',
      dataIndex: 'transaction_date',
      key: 'transaction_date',
      render: (date: string) => {
        if (!date) return '-';
        return new Date(date).toLocaleDateString();
      },
      sorter: (a: Transaction, b: Transaction) =>
        new Date(a.transaction_date || 0).getTime() - new Date(b.transaction_date || 0).getTime(),
      width: '12%',
    },
    {
      title: 'Posted Date',
      dataIndex: 'posted_date',
      key: 'posted_date',
      render: (date: string) => {
        if (!date) return '-';
        return new Date(date).toLocaleDateString();
      },
      sorter: (a: Transaction, b: Transaction) =>
        new Date(a.posted_date || 0).getTime() - new Date(b.posted_date || 0).getTime(),
      width: '12%',
    },
    {
      title: 'Card No.',
      dataIndex: 'card_number',
      key: 'card_number',
      render: (cardNumber: string) => cardNumber || '-',
      width: '10%',
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: '25%',
    },
    {
      title: 'Category',
      dataIndex: 'category_id',
      key: 'category_id',
      render: (categoryId: string) => {
        if (!categoryId) {
          return <Text type="secondary" italic>Uncategorized</Text>;
        }
        const category = categories.find(cat => cat.id === categoryId);
        return category ? (
          <Text style={{ color: category.color }}>
            {category.name}
          </Text>
        ) : (
          <Text type="secondary">Unknown Category</Text>
        );
      },
      width: '12%',
    },
    {
      title: 'Amount (Debit/Credit)',
      dataIndex: 'amount',
      key: 'amount',
      render: (amount: number) => {
        const isDebit = amount > 0;
        return (
          <div>
            <Text strong style={{ color: isDebit ? '#ff4d4f' : '#52c41a' }}>
              ${Math.abs(amount).toFixed(2)}
            </Text>
            <div style={{ fontSize: '11px', color: '#666' }}>
              {isDebit ? 'Debit' : 'Credit'}
            </div>
          </div>
        );
      },
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
      width: '12%',
    },
    {
      title: 'Action',
      key: 'action',
      render: (_: any, record: Transaction) => (
        <Select
          style={{ width: 120 }}
          placeholder="Assign"
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
      width: '12%',
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
          {/* Totals Section */}
          <Card
            title={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>
                  <DollarCircleOutlined style={{ marginRight: 8 }} />
                  Totals by Person
                </span>
                <Space.Compact>
                  <Input
                    placeholder="Enter person name"
                    value={newPersonName}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => setNewPersonName(e.target.value)}
                    onPressEnter={createPerson}
                    size="middle"
                    style={{ width: 200 }}
                  />
                  <Button
                    type="primary"
                    onClick={createPerson}
                    icon={<UserAddOutlined />}
                    size="middle"
                  >
                    Add Person
                  </Button>
                </Space.Compact>
              </div>
            }
            style={{ marginBottom: 24 }}
            variant='borderless'
            hoverable
          >
            {people.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '40px 0' }}>
                <Text type="secondary">
                  No people added yet. Add people to start tracking expenses.
                </Text>
              </div>
            ) : (
              <Row gutter={[16, 16]}>
                {people.map((person) => {
                  const personTotal = totals.find(t => t.person === person.name);
                  const totalAmount = personTotal ? personTotal.total : 0;

                  return (
                    <Col xs={12} sm={8} md={6} lg={4} key={person.id}>
                      <Card
                        size="small"
                        style={{ textAlign: 'center' }}
                        hoverable
                      >
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                          <Text strong style={{ fontSize: 14 }}>{person.name}</Text>
                          <Button
                            type="text"
                            danger
                            size="small"
                            icon={<DeleteOutlined />}
                            onClick={() => deletePerson(person.id, person.name)}
                          />
                        </div>
                        <Statistic
                          value={totalAmount}
                          precision={2}
                          prefix="$"
                          valueStyle={{
                            color: totalAmount > 0 ? '#3f8600' : totalAmount < 0 ? '#cf1322' : '#666666'
                          }}
                        />
                      </Card>
                    </Col>
                  );
                })}
              </Row>
            )}
          </Card>

          {/* Transactions Section */}
          <Card
            title={
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>
                  <FileTextOutlined style={{ marginRight: 8 }} />
                  Transactions ({transactions.length})
                </span>
                <Upload {...uploadProps}>
                  <Button
                    icon={<UploadOutlined />}
                    loading={uploading}
                    type="primary"
                    size="middle"
                  >
                    {uploading ? 'Uploading...' : 'Upload CSV'}
                  </Button>
                </Upload>
              </div>
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
                scroll={{ x: 1200 }}
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
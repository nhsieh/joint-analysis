import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
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
  Checkbox,
  Modal,
} from 'antd';
import {
  UploadOutlined,
  DollarCircleOutlined,
  FileTextOutlined,
  DeleteOutlined,
  ClearOutlined,
  PieChartOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import { Pie } from '@ant-design/charts';
import { UploadProps, RcFile } from 'antd/es/upload';
import { ColumnsType } from 'antd/es/table';

interface Transaction {
  id: string;
  description: string;
  amount: number;
  assigned_to: string[];
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
  email?: string;
  created_at: string;
  updated_at: string;
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

const { Text } = Typography;
const { Option } = Select;
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

const Dashboard: React.FC = () => {
  const [transactions, setTransactions] = useState<Transaction[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [totals, setTotals] = useState<PersonTotal[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [archiving, setArchiving] = useState(false);
  const [pageSize, setPageSize] = useState(50);
  const [currentPage, setCurrentPage] = useState(1);
  const [assignedFilter, setAssignedFilter] = useState<string>('all');

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

  // Define a consistent color palette for categories
  const getCategoryColor = (categoryName: string) => {
    // First, try to find the category in the database and use its color
    const category = categories.find(c => c.name === categoryName);
    if (category && category.color) {
      return category.color;
    }

    // Fallback color palette for categories not in database or without colors
    const colorPalette = [
      '#1f77b4', '#ff7f0e', '#2ca02c', '#d62728', '#9467bd',
      '#8c564b', '#e377c2', '#7f7f7f', '#bcbd22', '#17becf',
      '#aec7e8', '#ffbb78', '#98df8a', '#ff9896', '#c5b0d5',
      '#c49c94', '#f7b6d3', '#c7c7c7', '#dbdb8d', '#9edae5'
    ];

    // Create a hash of the category name to ensure consistent color assignment
    let hash = 0;
    for (let i = 0; i < categoryName.length; i++) {
      const char = categoryName.charCodeAt(i);
      hash = ((hash << 5) - hash) + char;
      hash = hash & hash; // Convert to 32bit integer
    }

    return colorPalette[Math.abs(hash) % colorPalette.length];
  };

  // Function to get pie chart data for a specific person
  const getPieChartData = (personName: string) => {
    // Filter transactions assigned to this person
    const personTransactions = transactions.filter(t =>
      (t.assigned_to || []).includes(personName) && t.amount > 0 // Only include debits (expenses)
    );

    // Group by category
    const categoryTotals: { [key: string]: number } = {};

    personTransactions.forEach(transaction => {
      const category = categories.find(c => c.id === transaction.category_id);
      const categoryName = category ? category.name : 'Uncategorized';

      // Split amount evenly among assigned people
      const assignedCount = transaction.assigned_to ? transaction.assigned_to.length : 1;
      const splitAmount = transaction.amount / assignedCount;

      categoryTotals[categoryName] = (categoryTotals[categoryName] || 0) + splitAmount;
    });

    // Convert to chart data format
    const chartData = Object.entries(categoryTotals)
      .map(([name, value]) => ({
        name,
        value,
        color: getCategoryColor(name),
      }))
      .sort((a, b) => b.value - a.value); // Sort by amount descending

    return chartData;
  };

  const handleFileUpload = async (file: RcFile): Promise<boolean> => {
    setUploading(true);
    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await axios.post(`${API_URL}/api/upload-csv`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      });

      const { transactions = [], skipped_rows = 0 } = response.data;
      const uploadedCount = transactions.length;

      if (skipped_rows > 0) {
        message.success(
          `CSV uploaded successfully! ${uploadedCount} transactions processed, ${skipped_rows} rows skipped.`
        );
      } else {
        message.success(`CSV uploaded successfully! ${uploadedCount} transactions processed.`);
      }

      await fetchTransactions();
      await fetchTotals();
      setCurrentPage(1); // Reset to first page after upload
    } catch (error) {
      console.error('Error uploading file:', error);
      message.error('Error uploading file. Please try again.');
    } finally {
      setUploading(false);
    }
    return false; // Prevent default upload behavior
  };

  const assignTransaction = async (transactionId: string, assignedPeopleUUIDs: string[]) => {
    try {
      await axios.put(`${API_URL}/api/transactions/${transactionId}/assign`, {
        assigned_to: assignedPeopleUUIDs,
      });
      fetchTransactions();
      fetchTotals();
    } catch (error) {
      console.error('Error assigning transaction:', error);
      message.error('Error assigning transaction');
    }
  };

  const clearAllTransactions = async () => {
    if (transactions.length === 0) {
      message.warning('No transactions to clear');
      return;
    }

    Modal.confirm({
      title: 'Clear All Transactions',
      content: `Are you sure you want to clear all ${transactions.length} transactions? This will not affect archived transactions. This action cannot be undone.`,
      okText: 'Yes, Clear All',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await axios.delete(`${API_URL}/api/transactions`);
          message.success('All transactions cleared successfully!');
          setCurrentPage(1); // Reset to first page after clearing
          fetchTransactions();
          fetchTotals();
        } catch (error) {
          console.error('Error clearing transactions:', error);
          message.error('Error clearing transactions');
        }
      },
    });
  };

  const archiveAllTransactions = async () => {
    if (transactions.length === 0) {
      message.warning('No transactions to archive');
      return;
    }

    try {
      setArchiving(true);
      await axios.post(`${API_URL}/api/archives`, {
        description: `Archived on ${new Date().toLocaleString()}`
      });
      message.success('Transactions archived successfully!');
      setCurrentPage(1); // Reset to first page after archiving
      fetchTransactions();
      fetchTotals();
    } catch (error) {
      console.error('Error archiving transactions:', error);
      message.error('Error archiving transactions');
    } finally {
      setArchiving(false);
    }
  };

  const deleteTransaction = async (transactionId: string) => {
    // Find the transaction to get its description for the confirmation message
    const transaction = transactions.find(t => t.id === transactionId);
    const transactionDescription = transaction ? transaction.description : 'this transaction';

    Modal.confirm({
      title: 'Delete Transaction',
      content: `Are you sure you want to delete "${transactionDescription}"? This action cannot be undone.`,
      okText: 'Yes, Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await axios.delete(`${API_URL}/api/transactions/${transactionId}`);
          message.success('Transaction deleted successfully!');
          fetchTransactions();
          fetchTotals();
        } catch (error) {
          console.error('Error deleting transaction:', error);
          message.error('Error deleting transaction');
        }
      },
    });
  };

  const updateTransactionCategory = async (transactionId: string, categoryId: string | null) => {
    try {
      await axios.put(`${API_URL}/api/transactions/${transactionId}/category`, {
        category_id: categoryId,
      });
      fetchTransactions();
    } catch (error) {
      console.error('Error updating transaction category:', error);
      message.error('Error updating transaction category');
    }
  };

  const uploadProps: UploadProps = {
    name: 'file',
    accept: '.csv',
    beforeUpload: handleFileUpload,
    showUploadList: false,
    multiple: false,
  };

  // Filter transactions based on assigned filter
  const filteredTransactions = assignedFilter === 'all'
    ? transactions
    : transactions.filter(transaction =>
        (transaction.assigned_to || []).includes(assignedFilter)
      );

  const columns: ColumnsType<Transaction> = [
    {
      title: 'Date',
      dataIndex: 'transaction_date',
      key: 'transaction_date',
      render: (date: string, record: Transaction) => {
        if (!date) return '-';
        // Parse date as YYYY-MM-DD and format as MM/DD/YYYY to avoid timezone issues
        const formatDateString = (dateStr: string): string => {
          const [year, month, day] = dateStr.split('-');
          return `${month}/${day}/${year}`;
        };

        const transactionDate = formatDateString(date);
        const postedDate = record.posted_date ? formatDateString(record.posted_date) : null;

        return (
          <div>
            <div>{transactionDate}</div>
            {postedDate && (
              <div style={{ fontSize: '11px', color: '#666', marginTop: 2 }}>
                Posted: {postedDate}
              </div>
            )}
          </div>
        );
      },
      sorter: (a: Transaction, b: Transaction) => {
        const dateA = a.transaction_date ? new Date(a.transaction_date + 'T12:00:00').getTime() : 0;
        const dateB = b.transaction_date ? new Date(b.transaction_date + 'T12:00:00').getTime() : 0;
        return dateA - dateB;
      },
      defaultSortOrder: 'descend',
      sortDirections: ['ascend', 'descend'],
      width: 100,
    },
    {
      title: 'Card',
      dataIndex: 'card_number',
      key: 'card_number',
      render: (cardNumber: string) => cardNumber || '-',
      width: 60,
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
      width: 250,
    },
    {
      title: 'Category',
      dataIndex: 'category_id',
      key: 'category_id',
      render: (categoryId: string, record: Transaction) => (
        <Select
          style={{ width: '100%' }}
          placeholder="Select category"
          value={categoryId || undefined}
          onChange={(value) => updateTransactionCategory(record.id, value || null)}
          allowClear
        >
          {categories.map((category) => (
            <Option key={category.id} value={category.id}>
              <span style={{ color: category.color }}>
                {category.name}
              </span>
            </Option>
          ))}
        </Select>
      ),
      width: 150,
    },
    {
      title: 'Amount',
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
      width: 100,
    },
    {
      title: 'Assign People',
      key: 'action',
      render: (_: any, record: Transaction) => (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
          {people.map((person) => {
            // Check if person is assigned by looking for their name in assigned_to array
            const isAssigned = (record.assigned_to || []).includes(person.name);

            return (
              <Checkbox
                key={person.id}
                checked={isAssigned}
                onChange={(e) => {
                  // Get current assigned person UUIDs
                  const currentAssignedNames = record.assigned_to || [];
                  let newAssignedUUIDs: string[];

                  if (e.target.checked) {
                    // Add this person's UUID to the list
                    const currentUUIDs = currentAssignedNames.map(name => {
                      const p = people.find(p => p.name === name);
                      return p ? p.id : '';
                    }).filter(id => id !== '');

                    newAssignedUUIDs = [...currentUUIDs, person.id];
                  } else {
                    // Remove this person's UUID from the list
                    const currentUUIDs = currentAssignedNames.map(name => {
                      const p = people.find(p => p.name === name);
                      return p ? p.id : '';
                    }).filter(id => id !== '');

                    newAssignedUUIDs = currentUUIDs.filter(uuid => uuid !== person.id);
                  }

                  assignTransaction(record.id, newAssignedUUIDs);
                }}
              >
                <span style={{ fontSize: '12px' }}>{person.name}</span>
              </Checkbox>
            );
          })}
        </div>
      ),
      width: 200,
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_: any, record: Transaction) => (
        <Button
          type="text"
          danger
          size="small"
          icon={<DeleteOutlined />}
          onClick={() => {
            deleteTransaction(record.id);
          }}
          title="Delete transaction"
        />
      ),
      width: 80,
    },
  ];

  return (
    <div style={{ padding: '24px', background: '#f0f2f5', minHeight: 'calc(100vh - 64px)' }}>
      <div style={{ maxWidth: 1200, margin: '0 auto' }}>
        {/* Totals Section */}
        <Card
          title={
            <span>
              <DollarCircleOutlined style={{ marginRight: 8 }} />
              Total Spent by Person
            </span>
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
            <Row gutter={[24, 16]} align="top">
              {/* Individual Totals Section */}
              <Col xs={24} lg={24} xl={18}>
                <Row gutter={[24, 24]}>
                  {people.map((person) => {
                    const personTotal = totals.find(t => t.person === person.name);
                    const totalAmount = personTotal ? personTotal.total : 0;
                    const chartData = getPieChartData(person.name);
                    const hasExpenses = chartData.length > 0 && chartData.some(d => d.value > 0);

                    return (
                      <Col xs={24} lg={12} xl={8} key={person.id}>
                        <Card
                          size="small"
                          hoverable
                          style={{ height: '100%', minHeight: '420px' }}
                        >
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
                            <Text strong style={{ fontSize: 18 }}>{person.name}</Text>
                          </div>

                          {/* Total Section - Centered Above Chart */}
                          <div style={{ textAlign: 'center', marginBottom: 24 }}>
                            <Statistic
                              value={totalAmount}
                              precision={2}
                              prefix="$"
                              valueStyle={{
                                color: totalAmount > 0 ? '#cf1322' : totalAmount < 0 ? '#3f8600': '#666666',
                                fontSize: '28px',
                                fontWeight: 'bold'
                              }}
                            />
                          </div>

                          {/* Pie Chart Section - Larger and More Prominent */}
                          <div style={{ width: '100%' }}>
                            {!hasExpenses ? (
                              <div style={{
                                textAlign: 'center',
                                padding: '60px 20px',
                                background: '#fafafa',
                                borderRadius: '8px',
                                border: '2px dashed #d9d9d9'
                              }}>
                                <PieChartOutlined style={{ fontSize: '48px', color: '#d9d9d9', marginBottom: '16px' }} />
                                <div>
                                  <Text type="secondary" style={{ fontSize: '16px', display: 'block' }}>
                                    No expenses assigned
                                  </Text>
                                  <Text type="secondary" style={{ fontSize: '14px' }}>
                                    Assign transactions to see spending breakdown
                                  </Text>
                                </div>
                              </div>
                            ) : (
                              <div style={{
                                minHeight: 400,
                                background: '#fafafa',
                                display: 'flex',
                                flexDirection: 'column',
                              }}>
                                {/* Pie Chart */}
                                <div style={{
                                  height: '320px', // Fixed height for consistent alignment
                                  display: 'flex',
                                  justifyContent: 'center',
                                  alignItems: 'center',
                                }}>
                                  <Pie
                                    key={`${person.name}-${JSON.stringify(chartData.map(item => ({ name: item.name, value: item.value })))}`}
                                    data={(() => {
                                      const total = chartData.reduce((sum, d) => sum + d.value, 0);
                                      return chartData.map(item => ({
                                        type: item.name, // Keep original name for pie chart
                                        value:  Number(item.value.toFixed(2)),
                                        originalName: item.name,
                                        color: item.color
                                      }));
                                    })()}
                                    angleField="value"
                                    colorField="type"
                                    radius={0.75}
                                    innerRadius={0.3}
                                    scale={{
                                      color: {
                                        relations: chartData.map(item => [item.name, item.color]),
                                      },
                                    }}
                                    legend={false}
                                    interactions={[
                                      { type: 'element-highlight' },
                                    ]}
                                  />
                                </div>

                                {/* Custom Legend */}
                                <div style={{
                                  display: 'flex',
                                  flexDirection: 'column',
                                  gap: '8px',
                                  padding: '16px' // Add padding all around legend
                                }}>
                                  {(() => {
                                    const total = chartData.reduce((sum, d) => sum + d.value, 0);
                                    return chartData.map((item, index) => (
                                      <div key={index} style={{ display: 'flex', alignItems: 'flex-start', gap: '8px' }}>
                                        <div
                                          style={{
                                            width: '12px',
                                            height: '12px',
                                            borderRadius: '50%',
                                            backgroundColor: item.color,
                                            marginTop: '2px',
                                            flexShrink: 0
                                          }}
                                        />
                                        <div style={{ fontSize: '12px', lineHeight: '1.2', flex: 1 }}>
                                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                            <span style={{ fontWeight: 500, color: '#333' }}>
                                              {item.name} ({((item.value / total) * 100).toFixed(1)}%)
                                            </span>
                                            <span style={{ color: '#666', fontSize: '11px' }}>
                                              ${item.value.toFixed(2)}
                                            </span>
                                          </div>
                                        </div>
                                      </div>
                                    ));
                                  })()}
                                </div>
                                </div>
                            )}
                          </div>
                        </Card>
                      </Col>
                    );
                  })}
                </Row>
              </Col>

              {/* Grand Total Section */}
              <Col xs={24} lg={24} xl={6}>
                <Card
                  size="small"
                  style={{
                    textAlign: 'center',
                    background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
                    border: 'none',
                    height: 'fit-content',
                    position: 'sticky',
                    top: '24px'
                  }}
                  hoverable
                >
                  <div style={{ color: 'white' }}>
                    <Text style={{ color: 'white', fontSize: 16, fontWeight: 500 }}>
                      Grand Total Spent
                    </Text>
                    <div style={{ marginTop: 8 }}>
                      <Statistic
                        value={totals.reduce((sum, total) => sum + total.total, 0)}
                        precision={2}
                        prefix="$"
                        valueStyle={{
                          color: 'white',
                          fontSize: 32,
                          fontWeight: 'bold'
                        }}
                      />
                    </div>
                  </div>
                </Card>
              </Col>
            </Row>
          )}
        </Card>

        {/* Transactions Section */}
        <Card
          title={
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <span>
                <FileTextOutlined style={{ marginRight: 8 }} />
                Transactions ({filteredTransactions.length})
              </span>
              <div style={{ display: 'flex', gap: '8px' }}>
                <Select
                  style={{ width: 200 }}
                  placeholder="Filter by person"
                  value={assignedFilter}
                  onChange={(value) => {
                    setAssignedFilter(value);
                    setCurrentPage(1); // Reset to first page when filtering
                  }}
                >
                  <Option value="all">All Transactions</Option>
                  {people.map((person) => (
                    <Option key={person.id} value={person.name}>
                      {person.name}
                    </Option>
                  ))}
                </Select>
                <Button
                  icon={<InboxOutlined />}
                  size="middle"
                  onClick={archiveAllTransactions}
                  disabled={transactions.length === 0}
                  loading={archiving}
                >
                  Archive
                </Button>
                <Button
                  icon={<ClearOutlined />}
                  danger
                  size="middle"
                  onClick={clearAllTransactions}
                  disabled={transactions.length === 0}
                >
                  Clear All
                </Button>
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
            </div>
          }
          variant='borderless'
          hoverable
        >
          <Spin spinning={loading}>
            <Table
              columns={columns}
              dataSource={filteredTransactions}
              rowKey="id"
              pagination={{
                current: currentPage,
                pageSize: pageSize,
                showSizeChanger: true,
                showQuickJumper: false,
                showTotal: (total: number, range: [number, number]) =>
                  `${range[0]}-${range[1]} of ${total} transactions`,
                pageSizeOptions: ['10', '20', '50', '100'],
                onChange: (page: number, size: number) => {
                  setCurrentPage(page);
                  setPageSize(size);
                },
                onShowSizeChange: (current: number, size: number) => {
                  setCurrentPage(1); // Reset to first page when changing page size
                  setPageSize(size);
                },
              }}
              scroll={{ x: 910 }}
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
    </div>
  );
};

export default Dashboard;
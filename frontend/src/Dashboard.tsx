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
  Row,
  Col,
  Statistic,
  message,
  Spin,
  Checkbox,
  Modal,
  InputNumber,
} from 'antd';
import {
  UploadOutlined,
  DollarCircleOutlined,
  FileTextOutlined,
  DeleteOutlined,
  SwapOutlined,
  ClearOutlined,
  PieChartOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import { Pie } from '@ant-design/charts';
import { UploadProps, RcFile } from 'antd/es/upload';
import { ColumnsType } from 'antd/es/table';
import { Transaction, Person, Category, PersonTotal, TransactionSplit } from './types';
import { getCategoryColor, generateColorVariants } from './utils';

const { Text } = Typography;
const { Option } = Select;
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081';

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
  const [splitModalOpen, setSplitModalOpen] = useState(false);
  const [splitSaving, setSplitSaving] = useState(false);
  const [splitRows, setSplitRows] = useState<TransactionSplit[]>([]);
  const [splitTransaction, setSplitTransaction] = useState<Transaction | null>(null);
  // drillDownState maps personName → top-level category ID being drilled into (null = top-level view)
  const [drillDownState, setDrillDownState] = useState<Record<string, string | null>>({});

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

  // Flatten nested category tree into a flat list for lookups
  const flatCategories = React.useMemo(() => {
    const flat: Category[] = [];
    const flatten = (cats: Category[]) => {
      cats.forEach(c => {
        flat.push(c);
        if (c.subcategories) flatten(c.subcategories);
      });
    };
    flatten(categories);
    return flat;
  }, [categories]);

  // Function to get pie chart data for a specific person.
  // If drillDownCategoryId is set, show subcategory breakdown for that top-level category.
  // Otherwise, aggregate by top-level category (subcategory transactions roll up to parent).
  const getPieChartData = (personName: string, drillDownCategoryId?: string | null) => {
    const personTransactions = transactions.filter(t =>
      (t.assigned_to || []).includes(personName)
    );

    const categoryTotals: { [key: string]: number } = {};

    personTransactions.forEach(transaction => {
      const assignedCount = transaction.assigned_to ? transaction.assigned_to.length : 1;
      const txSign = transaction.amount < 0 ? -1 : 1;

      const categoryAllocations = (transaction.splits && transaction.splits.length > 0)
        ? transaction.splits.map(split => ({
            categoryId: split.category_id,
            amount: txSign * Number(split.amount || 0),
          }))
        : [];

      categoryAllocations.forEach(allocation => {
        const txCategory = allocation.categoryId
          ? flatCategories.find(c => c.id === allocation.categoryId)
          : undefined;
        const amountPerPerson = allocation.amount / assignedCount;

        if (drillDownCategoryId) {
          // Drill-down: only include allocations in this top-level category or its subcategories
          if (txCategory) {
            const isTopLevel = txCategory.id === drillDownCategoryId;
            const isSubOfTarget = txCategory.parent_id === drillDownCategoryId;
            if (isTopLevel || isSubOfTarget) {
              const label = txCategory.name;
              categoryTotals[label] = (categoryTotals[label] || 0) + amountPerPerson;
            }
          }
        } else {
          // Top-level view: roll subcategory amounts up to their parent
          if (!txCategory) {
            categoryTotals['Uncategorized'] = (categoryTotals['Uncategorized'] || 0) + amountPerPerson;
          } else if (txCategory.parent_id) {
            const parent = flatCategories.find(c => c.id === txCategory.parent_id);
            const parentName = parent ? parent.name : txCategory.name;
            categoryTotals[parentName] = (categoryTotals[parentName] || 0) + amountPerPerson;
          } else {
            categoryTotals[txCategory.name] = (categoryTotals[txCategory.name] || 0) + amountPerPerson;
          }
        }
      });
    });

    const sorted = Object.entries(categoryTotals)
      .map(([name, value]) => ({ name, value }))
      .sort((a, b) => Math.abs(b.value) - Math.abs(a.value));

    if (drillDownCategoryId) {
      // Generate distinct color variants from the parent's base color
      const parentCat = flatCategories.find(c => c.id === drillDownCategoryId);
      const baseColor = parentCat?.color || '#1890ff';
      const variants = generateColorVariants(baseColor, sorted.length);
      return sorted.map((item, i) => ({ ...item, color: variants[i] }));
    }

    return sorted.map(item => ({
      ...item,
      color: getCategoryColor(item.name, categories),
    }));
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

  const openSplitModal = async (transaction: Transaction) => {
    try {
      const response = await axios.get(`${API_URL}/api/transactions/${transaction.id}/splits`);
      const rows: TransactionSplit[] = (response.data || []).map((row: any) => ({
        id: row.id,
        transaction_id: row.transaction_id,
        amount: Number(row.amount || 0),
        category_id: row.category_id || '',
        notes: row.notes || '',
      }));

      setSplitTransaction(transaction);
      setSplitRows(rows);
      setSplitModalOpen(true);
    } catch (error) {
      console.error('Error loading transaction splits:', error);
      message.error('Error loading transaction splits');
    }
  };

  const addSplitRow = () => {
    const fallbackCategory = flatCategories[0]?.id || '';
    setSplitRows(prev => {
      const expectedTotal = Math.abs(splitTransaction?.amount || 0);
      const currentTotal = prev.reduce((sum, row) => sum + Number(row.amount || 0), 0);
      const remaining = Math.max(0, Number((expectedTotal - currentTotal).toFixed(2)));

      return [...prev, { amount: remaining, category_id: fallbackCategory, notes: '' }];
    });
  };

  const updateSplitRow = (index: number, patch: Partial<TransactionSplit>) => {
    setSplitRows(prev => prev.map((row, i) => (i === index ? { ...row, ...patch } : row)));
  };

  const removeSplitRow = (index: number) => {
    setSplitRows(prev => prev.filter((_, i) => i !== index));
  };

  const getSplitValidationError = (): string | null => {
    if (!splitTransaction) return 'No transaction selected';
    if (splitRows.length === 0) return 'At least one split row is required';

    for (const row of splitRows) {
      if (!row.category_id) return 'Each split row must have a category';
      if (!row.amount || row.amount <= 0) return 'Each split amount must be greater than zero';
    }

    const splitTotal = splitRows.reduce((sum, row) => sum + Number(row.amount || 0), 0);
    const expected = Math.abs(splitTransaction.amount || 0);
    if (Math.abs(splitTotal - expected) > 0.01) {
      return `Split total must equal $${expected.toFixed(2)}`;
    }

    return null;
  };

  const saveSplits = async () => {
    if (!splitTransaction) return;

    const validationError = getSplitValidationError();
    if (validationError) {
      message.error(validationError);
      return;
    }

    try {
      setSplitSaving(true);
      await axios.put(`${API_URL}/api/transactions/${splitTransaction.id}/splits`, {
        splits: splitRows.map(row => ({
          amount: Number(Number(row.amount || 0).toFixed(2)),
          category_id: row.category_id,
          notes: row.notes || undefined,
        })),
      });

      message.success('Transaction splits updated');
      setSplitModalOpen(false);
      setSplitTransaction(null);
      setSplitRows([]);
      await fetchTransactions();
    } catch (error) {
      console.error('Error saving transaction splits:', error);
      message.error('Error saving transaction splits');
    } finally {
      setSplitSaving(false);
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

  const getCategoryNameByID = (categoryID: string): string => {
    const category = flatCategories.find(c => c.id === categoryID);
    if (!category) return 'Unknown Category';

    if (category.parent_id) {
      const parent = flatCategories.find(c => c.id === category.parent_id);
      return parent ? `${parent.name} / ${category.name}` : category.name;
    }

    return category.name;
  };

  const getCategoryColorByID = (categoryID: string): string | undefined => {
    const category = flatCategories.find(c => c.id === categoryID);
    if (!category) return undefined;

    if (category.parent_id) {
      const parent = flatCategories.find(c => c.id === category.parent_id);
      return parent?.color || category.color;
    }

    return category.color;
  };

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
      dataIndex: 'splits',
      key: 'category_id',
      render: (_: TransactionSplit[] | undefined, record: Transaction) => {
        return (
          <div style={{ display: 'flex', alignItems: 'flex-start', gap: 8, width: '100%' }}>
            <div style={{ fontSize: 11, color: '#666', flex: 1, minWidth: 0 }}>
              {(record.splits || []).map((split, idx) => (
                <div key={`${record.id}-split-${idx}`}>
                  <span style={{ color: getCategoryColorByID(split.category_id) || '#666' }}>
                    {getCategoryNameByID(split.category_id)}
                  </span>
                  : ${Number(split.amount || 0).toFixed(2)}
                </div>
              ))}
            </div>
            <Button
              type="text"
              size="small"
              icon={<SwapOutlined />}
              onClick={() => openSplitModal(record)}
              title="Edit splits"
            />
          </div>
        );
      },
      width: 180,
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
          extra={
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <Text style={{ fontSize: 14, color: '#666' }}>Grand Total Spent:</Text>
              <Statistic
                value={totals.reduce((sum, total) => sum + total.total, 0)}
                precision={2}
                prefix="$"
                valueStyle={{
                  color: '#cf1322',
                  fontSize: 24,
                  fontWeight: 'bold'
                }}
              />
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
            <Row gutter={[24, 16]} align="top">
              {/* Individual Totals Section */}
              <Col xs={24}>
                <Row gutter={[24, 24]}>
                  {people.map((person) => {
                    const drillDownCategoryId = drillDownState[person.name] || null;
                    const personTotal = totals.find(t => t.person === person.name);
                    const totalAmount = personTotal ? personTotal.total : 0;
                    const chartData = getPieChartData(person.name, drillDownCategoryId);
                    const hasTransactions = chartData.length > 0 && chartData.some(d => d.value !== 0);

                    // Find the top-level category for the current drill-down
                    const drillDownCategory = drillDownCategoryId
                      ? categories.find(c => c.id === drillDownCategoryId)
                      : null;

                    return (
                      <Col xs={24} lg={12} xl={8} key={person.id}>
                        <Card
                          size="small"
                          hoverable
                          style={{ height: '100%', minHeight: '420px' }}
                        >
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
                            <div>
                              <Text strong style={{ fontSize: 18 }}>{person.name}</Text>
                              {drillDownCategory && (
                                <div style={{ marginTop: 4 }}>
                                  <Button
                                    size="small"
                                    type="link"
                                    onClick={() => setDrillDownState(prev => ({ ...prev, [person.name]: null }))}
                                    style={{ padding: 0, fontSize: 12 }}
                                  >
                                    ← Back
                                  </Button>
                                  <Text type="secondary" style={{ fontSize: 12, marginLeft: 4 }}>
                                    {drillDownCategory.name} breakdown
                                  </Text>
                                </div>
                              )}
                            </div>
                            <Statistic
                              value={totalAmount}
                              precision={2}
                              prefix="$"
                              valueStyle={{
                                color: totalAmount > 0 ? '#cf1322' : totalAmount < 0 ? '#3f8600': '#666666',
                                fontSize: '24px',
                                fontWeight: 'bold'
                              }}
                            />
                          </div>

                          {/* Pie Chart Section - Larger and More Prominent */}
                          <div style={{ width: '100%' }}>
                            {!hasTransactions ? (
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
                                    key={`${person.name}-${drillDownCategoryId || 'top'}-${JSON.stringify(chartData.map(item => ({ name: item.name, value: item.value })))}`}
                                    data={(() => {
                                      return chartData
                                        .filter(item => item.value !== 0)
                                        .map(item => ({
                                          type: item.name,
                                          value: Number(Math.abs(item.value).toFixed(2)),
                                          originalValue: item.value,
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
                                    // Calculate total based on absolute values for percentage
                                    const totalAbsolute = chartData.reduce((sum, d) => sum + Math.abs(d.value), 0);
                                    return chartData
                                      .filter(item => item.value !== 0)
                                      .map((item, index) => {
                                        const topCat = !drillDownCategoryId
                                          ? categories.find(c => c.name === item.name)
                                          : null;
                                        const isDrillable = topCat && topCat.subcategories && topCat.subcategories.length > 0;
                                        return (
                                          <div
                                            key={index}
                                            onClick={() => {
                                              if (isDrillable) {
                                                setDrillDownState(prev => ({ ...prev, [person.name]: topCat!.id }));
                                              }
                                            }}
                                            style={{
                                              display: 'flex',
                                              alignItems: 'flex-start',
                                              gap: '8px',
                                              cursor: isDrillable ? 'pointer' : 'default',
                                              borderRadius: 4,
                                              padding: '2px 4px',
                                              margin: '-2px -4px',
                                            }}
                                            title={isDrillable ? `Click to drill into ${item.name}` : undefined}
                                          >
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
                                                <span style={{ fontWeight: 500, color: isDrillable ? '#1890ff' : '#333', textDecoration: isDrillable ? 'underline' : 'none' }}>
                                                  {item.name} ({((Math.abs(item.value) / totalAbsolute) * 100).toFixed(1)}%)
                                                </span>
                                                <span style={{
                                                  color: item.value < 0 ? '#3f8600' : '#666',
                                                  fontSize: '11px',
                                                  fontWeight: item.value < 0 ? 600 : 400
                                                }}>
                                                  {item.value < 0 ? '-' : ''}${Math.abs(item.value).toFixed(2)}
                                                </span>
                                              </div>
                                            </div>
                                          </div>
                                        );
                                      });
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

        <Modal
          title={splitTransaction ? `Split: ${splitTransaction.description}` : 'Split Transaction'}
          open={splitModalOpen}
          onCancel={() => {
            setSplitModalOpen(false);
            setSplitTransaction(null);
            setSplitRows([]);
          }}
          onOk={saveSplits}
          okText="Save Splits"
          confirmLoading={splitSaving}
          width={900}
        >
          <div style={{ marginBottom: 12, color: '#666' }}>
            Transaction total: <strong>${Math.abs(splitTransaction?.amount || 0).toFixed(2)}</strong>
          </div>

          {splitRows.map((row, index) => (
            <Row key={index} gutter={8} style={{ marginBottom: 8 }} align="middle">
              <Col span={9}>
                <Select
                  value={row.category_id || undefined}
                  placeholder="Category"
                  style={{ width: '100%' }}
                  onChange={(value) => updateSplitRow(index, { category_id: value })}
                  showSearch
                  optionFilterProp="label"
                >
                  {categories.map((category) => {
                    const subs = category.subcategories || [];
                    if (subs.length > 0) {
                      return [
                        <Option key={category.id} value={category.id} label={category.name}>
                          <span style={{ color: category.color, fontWeight: 600 }}>{category.name}</span>
                        </Option>,
                        ...subs.map((sub) => (
                          <Option key={sub.id} value={sub.id} label={`${category.name} / ${sub.name}`}>
                            <span style={{ color: category.color, paddingLeft: 8 }}>↳ {sub.name}</span>
                          </Option>
                        )),
                      ];
                    }
                    return (
                      <Option key={category.id} value={category.id} label={category.name}>
                        <span style={{ color: category.color }}>{category.name}</span>
                      </Option>
                    );
                  })}
                </Select>
              </Col>

              <Col span={5}>
                <InputNumber
                  min={0}
                  step={0.01}
                  precision={2}
                  style={{ width: '100%' }}
                  value={row.amount}
                  onChange={(value) => updateSplitRow(index, { amount: Number(value || 0) })}
                />
              </Col>

              <Col span={8}>
                <Input
                  placeholder="Notes (optional)"
                  value={row.notes || ''}
                  onChange={(e) => updateSplitRow(index, { notes: e.target.value })}
                />
              </Col>

              <Col span={2}>
                <Button
                  danger
                  onClick={() => removeSplitRow(index)}
                  disabled={splitRows.length <= 1}
                >
                  X
                </Button>
              </Col>
            </Row>
          ))}

          <Row>
            <Col>
              <Button onClick={addSplitRow}>+ Add Split Row</Button>
            </Col>
          </Row>

          <div style={{ marginTop: 12, color: getSplitValidationError() ? '#cf1322' : '#389e0d' }}>
            {getSplitValidationError()
              ? getSplitValidationError()
              : `Split total: $${splitRows.reduce((sum, row) => sum + Number(row.amount || 0), 0).toFixed(2)}`}
          </div>
        </Modal>
      </div>
    </div>
  );
};

export default Dashboard;
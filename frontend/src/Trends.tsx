import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Typography,
  Card,
  Row,
  Col,
  message,
  Spin,
  Select,
} from 'antd';
import {
  LineChartOutlined,
} from '@ant-design/icons';
import { Line, Pie } from '@ant-design/charts';
import { Archive, Transaction, Person, Category, PersonTotal } from './types';
import { getCategoryColor } from './utils';

interface CategorySpendingData {
  archive: string;
  person: string;
  amount: number;
  category: string;
}

interface TotalSpendingData {
  archive: string;
  person: string;
  total: number;
}

interface SpendingBalanceData {
  archive: string;
  person: string;
  balance: number;
}

const { Title } = Typography;
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

const Trends: React.FC = () => {
  const [archives, setArchives] = useState<Archive[]>([]);
  const [people, setPeople] = useState<Person[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedArchive, setSelectedArchive] = useState<string | null>(null);

  const [categorySpendingData, setCategorySpendingData] = useState<CategorySpendingData[]>([]);
  const [totalSpendingData, setTotalSpendingData] = useState<TotalSpendingData[]>([]);
  const [balanceData, setBalanceData] = useState<SpendingBalanceData[]>([]);
  const [topCategoriesData, setTopCategoriesData] = useState<any[]>([]);

  useEffect(() => {
    fetchData();
  }, []);

  useEffect(() => {
    if (archives.length > 0 && people.length > 0 && categories.length > 0) {
      processChartData();
      // Set default selected archive to the most recent one
      if (!selectedArchive && archives.length > 0) {
        const sortedArchives = [...archives].sort(
          (a, b) => new Date(b.archived_at).getTime() - new Date(a.archived_at).getTime()
        );
        const mostRecent = sortedArchives[0];
        const archiveLabel = mostRecent.description || new Date(mostRecent.archived_at).toLocaleDateString();
        setSelectedArchive(archiveLabel);
      }
    }
  }, [archives, people, categories]);

  const fetchData = async () => {
    try {
      setLoading(true);
      const [archivesRes, peopleRes, categoriesRes] = await Promise.all([
        axios.get(`${API_URL}/api/archives`),
        axios.get(`${API_URL}/api/people`),
        axios.get(`${API_URL}/api/categories`),
      ]);

      setArchives(archivesRes.data || []);
      setPeople(peopleRes.data || []);
      setCategories(categoriesRes.data || []);
    } catch (error) {
      console.error('Error fetching data:', error);
      message.error('Error fetching data');
    } finally {
      setLoading(false);
    }
  };

  const processChartData = async () => {
    const categorySpending: CategorySpendingData[] = [];
    const totalSpending: TotalSpendingData[] = [];
    const topCategories: Map<string, Map<string, number>> = new Map(); // archive -> category -> total

    // Sort archives by date
    const sortedArchives = [...archives].sort(
      (a, b) => new Date(a.archived_at).getTime() - new Date(b.archived_at).getTime()
    );

    for (const archive of sortedArchives) {
      const archiveLabel = archive.description || new Date(archive.archived_at).toLocaleDateString();

      try {
        // Fetch transactions for this archive
        const response = await axios.get(`${API_URL}/api/archives/${archive.id}/transactions`);
        const transactions: Transaction[] = response.data || [];

        // Initialize category map for this archive
        topCategories.set(archiveLabel, new Map());

        // Process each transaction
        for (const transaction of transactions) {
          const category = categories.find(c => c.id === transaction.category_id);
          const categoryName = category?.name || 'Uncategorized';

          // Skip Reimbursable category
          if (categoryName === 'Reimbursable') {
            continue;
          }

          // Track for top categories
          const archiveCategoryMap = topCategories.get(archiveLabel)!;
          archiveCategoryMap.set(
            categoryName,
            (archiveCategoryMap.get(categoryName) || 0) + transaction.amount
          );

          // Calculate amount per person for this transaction
          // Note: assigned_to contains person NAMES, not IDs
          const assignedPeople = transaction.assigned_to && transaction.assigned_to.length > 0
            ? transaction.assigned_to
            : people.map(p => p.name); // If no assignment, assign to all people

          const numPeople = assignedPeople.length;
          const amountPerPerson = transaction.amount / numPeople;

          for (const personName of assignedPeople) {
            const person = people.find(p => p.name === personName);
            if (!person) {
              continue;
            }

            // Add to category spending data
            categorySpending.push({
              archive: archiveLabel,
              person: person.name,
              amount: amountPerPerson,
              category: categoryName,
            });
          }
        }

        // Calculate total spending per person from filtered transactions (excluding Reimbursable)
        const personTotals = new Map<string, number>();

        for (const transaction of transactions) {
          const category = categories.find(c => c.id === transaction.category_id);
          const categoryName = category?.name || 'Uncategorized';

          // Skip Reimbursable category for totals
          if (categoryName === 'Reimbursable') {
            continue;
          }

          const assignedPeople = transaction.assigned_to && transaction.assigned_to.length > 0
            ? transaction.assigned_to
            : people.map(p => p.name);

          const numPeople = assignedPeople.length;
          const amountPerPerson = transaction.amount / numPeople;

          for (const personName of assignedPeople) {
            personTotals.set(personName, (personTotals.get(personName) || 0) + amountPerPerson);
          }
        }

        // Add to total spending data
        personTotals.forEach((total, personName) => {
          totalSpending.push({
            archive: archiveLabel,
            total: parseFloat(total.toFixed(2)),
            person: personName,
          });
        });
      } catch (error) {
        console.error(`Error processing archive ${archive.id}:`, error);
      }
    }

    // Aggregate category spending by person, category, and archive
    const aggregatedCategorySpending = categorySpending.reduce((acc, curr) => {
      const key = `${curr.archive}-${curr.category}-${curr.person}`;
      if (!acc[key]) {
        acc[key] = { ...curr };
      } else {
        acc[key].amount = parseFloat((acc[key].amount + curr.amount).toFixed(2));
      }
      return acc;
    }, {} as Record<string, CategorySpendingData>);

    setCategorySpendingData(Object.values(aggregatedCategorySpending));
    setTotalSpendingData(totalSpending);

    // Calculate balance (difference from average)
    calculateBalance(totalSpending);

    // Process top categories
    const topCatData: any[] = [];
    topCategories.forEach((categoryMap, archive) => {
      categoryMap.forEach((amount, category) => {
        topCatData.push({
          archive,
          category,
          amount: parseFloat(amount.toFixed(2)),
        });
      });
    });
    setTopCategoriesData(topCatData);
  };

  const calculateBalance = (totalData: TotalSpendingData[]) => {
    // Group by archive
    const byArchive = totalData.reduce((acc, curr) => {
      if (!acc[curr.archive]) {
        acc[curr.archive] = [];
      }
      acc[curr.archive].push(curr);
      return acc;
    }, {} as Record<string, TotalSpendingData[]>);

    const balances: SpendingBalanceData[] = [];

    Object.entries(byArchive).forEach(([archive, data]) => {
      const average = data.reduce((sum, d) => sum + d.total, 0) / data.length;
      data.forEach(d => {
        balances.push({
          archive,
          person: d.person,
          balance: d.total - average,
        });
      });
    });

    setBalanceData(balances);
  };

  // Get unique archive labels for selector
  const archiveLabels: string[] = Array.from(new Set(categorySpendingData.map(d => d.archive))).sort();



  // Filter and group data for pie charts by person
  const getPieDataByPerson = () => {
    if (!selectedArchive) return [];

    // Filter data for selected archive
    const filteredData = categorySpendingData.filter(d => d.archive === selectedArchive);

    // Group by person
    const byPerson = filteredData.reduce((acc, curr) => {
      if (!acc[curr.person]) {
        acc[curr.person] = [];
      }

      // Check if category already exists for this person
      const existingCategory = acc[curr.person].find(item => item.category === curr.category);
      if (existingCategory) {
        existingCategory.amount += curr.amount;
      } else {
        acc[curr.person].push({
          category: curr.category,
          amount: curr.amount,
        });
      }
      return acc;
    }, {} as Record<string, Array<{ category: string; amount: number }>>);

    return Object.entries(byPerson)
      .map(([person, data]) => ({
        person,
        data: data
          .map(d => ({
            type: d.category,
            value: parseFloat(d.amount.toFixed(2)),
          }))
          .sort((a, b) => b.value - a.value), // Sort by value descending
      }))
      .sort((a, b) => {
        // "Joint" comes first
        if (a.person === 'Joint') return -1;
        if (b.person === 'Joint') return 1;
        // Then alphabetically
        return a.person.localeCompare(b.person);
      });
  };

  const pieDataByPerson = getPieDataByPerson();

  // Chart configurations
  const totalSpendingConfig = {
    data: totalSpendingData,
    xField: 'archive',
    yField: 'total',
    colorField: 'person',
    point: {
      size: 5,
      shape: 'circle',
    },
  };

  const topCategoriesConfig = {
    data: topCategoriesData,
    xField: 'archive',
    yField: 'amount',
    colorField: 'category',
    scale: {
      color: {
        range: Array.from(new Set(topCategoriesData.map(d => d.category)))
          .map(cat => getCategoryColor(cat, categories)),
      },
    },
    point: {
      size: 5,
      shape: 'circle',
    },
  };

  if (loading) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (archives.length === 0) {
    return (
      <div style={{ padding: 24, textAlign: 'center' }}>
        <Title level={3}>No archives available</Title>
        <p>Create archives from the Dashboard to see trends</p>
      </div>
    );
  }

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={[16, 16]} align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <Title level={2} style={{ margin: 0 }}>
            <LineChartOutlined /> Spending Trends
          </Title>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col span={24}>
          <Card title="Total Spending Over Time">
            <Line {...totalSpendingConfig} />
          </Card>
        </Col>

        <Col span={24}>
          <Card title="Category Distribution Over Time">
            <Line {...topCategoriesConfig} />
          </Card>
        </Col>

        <Col span={24}>
          <Card
            title="Category Spending by Person"
            extra={
              <Select
                style={{ width: 300 }}
                placeholder="Select archive"
                value={selectedArchive}
                onChange={setSelectedArchive}
              >
                {archiveLabels.map(label => (
                  <Select.Option key={label} value={label}>
                    {label}
                  </Select.Option>
                ))}
              </Select>
            }
          >
            {pieDataByPerson.length === 0 ? (
              <p style={{ textAlign: 'center', color: '#999' }}>
                Select an archive to view category breakdown by person
              </p>
            ) : (
              <Row gutter={[16, 16]}>
                {pieDataByPerson.map(({ person, data }) => {
                  const personTotal = data.reduce((sum, d) => sum + d.value, 0);
                  return (
                    <Col xs={24} sm={24} md={12} lg={8} key={person}>
                      <Card
                        size="small"
                        title={
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <span>{person}</span>
                            <span style={{ fontWeight: 'normal', color: '#666' }}>
                              ${personTotal.toFixed(2)}
                            </span>
                          </div>
                        }
                      >
                      <div>
                        {/* Pie Chart */}
                        <div style={{
                          height: '320px',
                          display: 'flex',
                          justifyContent: 'center',
                          alignItems: 'center',
                        }}>
                          <Pie
                            key={`${person}-${selectedArchive}-${JSON.stringify(data)}`}
                            data={data.map(d => ({
                              ...d,
                              color: getCategoryColor(d.type, categories),
                            }))}
                            angleField="value"
                            colorField="type"
                            radius={0.75}
                            innerRadius={0.3}
                            scale={{
                              color: {
                                range: data.map(d => getCategoryColor(d.type, categories)),
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
                          padding: '16px'
                        }}>
                          {(() => {
                            const total = data.reduce((sum, d) => sum + d.value, 0);
                            return data.map((item, index) => (
                              <div key={index} style={{ display: 'flex', alignItems: 'flex-start', gap: '8px' }}>
                                <div
                                  style={{
                                    width: '12px',
                                    height: '12px',
                                    borderRadius: '50%',
                                    backgroundColor: getCategoryColor(item.type, categories),
                                    marginTop: '2px',
                                    flexShrink: 0
                                  }}
                                />
                                <div style={{ fontSize: '12px', lineHeight: '1.2', flex: 1 }}>
                                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                    <span style={{ fontWeight: 500, color: '#333' }}>
                                      {item.type} ({((item.value / total) * 100).toFixed(1)}%)
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
                    </Card>
                  </Col>
                  );
                })}
              </Row>
            )}
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Trends;

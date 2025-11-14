import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Typography,
  Card,
  Row,
  Col,
  message,
  Spin,
} from 'antd';
import {
  LineChartOutlined,
} from '@ant-design/icons';
import { Line, Column, Area } from '@ant-design/charts';

interface Archive {
  id: string;
  description?: string;
  archived_at: string;
  transaction_count: number;
  total_amount: number;
  person_totals?: PersonTotal[];
  created_at: string;
  updated_at: string;
}

interface PersonTotal {
  name: string;
  total: number;
}

interface Transaction {
  id: string;
  description: string;
  amount: number;
  assigned_to: string[];
  category_id: string;
  transaction_date: string;
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

  // Chart configurations
  const categorySpendingConfig = {
    // data: categorySpendingData.map(item => ({
    //   ...item,
    //   archivePerson: `${item.archive} - ${item.person}`,
    // })),
    // xField: 'archivePerson',
    // yField: 'amount',
    // colorField: 'category',
    data: categorySpendingData,
    xField: 'archive',
    yField: 'amount',
    seriesField: 'person',
    stack: {
      groupBy: ['x', 'series'],
      series: false,
    },
    colorField: 'category',
    label: {
      text: (d: any) => `[${d.person}] ${d.category}`,
      position: 'top',
      style: {
        fontSize: 10,
        fontWeight: 'bold',
      },
    },
    tooltip: (item: any) => {
      return { origin: item };
    },
    interaction: {
      tooltip: {
        render: (e: any, { title, items }: { title: string; items: any[] }) => {
          return (
            <div>
              <h4>{title}</h4>
              {items.map((item: any) => {
                const { name, color, origin } = item;
                return (
                  <div>
                    <div style={{ margin: 0, display: 'flex', justifyContent: 'space-between' }}>
                      <div>
                        <span
                          style={{
                            display: 'inline-block',
                            width: 6,
                            height: 6,
                            borderRadius: '50%',
                            backgroundColor: color,
                            marginRight: 6,
                          }}
                        ></span>
                        <span>
                          [{name}] {origin['category']}
                        </span>
                      </div>
                      <b>{origin['amount']}</b>
                    </div>
                  </div>
                );
              })}
            </div>
          );
        },
      },
    },
  };

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
          <Card title="Category Spending by Person">
              <Column {...categorySpendingConfig} />
          </Card>
        </Col>

        <Col span={24}>
          <Card title="Category Distribution Over Time">
            <Line {...topCategoriesConfig} />
          </Card>
        </Col>
      </Row>
    </div>
  );
};

export default Trends;

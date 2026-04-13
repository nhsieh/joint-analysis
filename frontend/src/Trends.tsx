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
  InfoCircleOutlined,
} from '@ant-design/icons';
import { Line, Pie } from '@ant-design/charts';
import { Archive, Transaction, Person, Category, PersonTotal } from './types';
import { getCategoryColor, generateColorVariants } from './utils';

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
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081';

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
  // drillDownState maps personName → top-level category name being drilled into
  const [drillDownState, setDrillDownState] = useState<Record<string, string | null>>({});

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

    // Flatten the nested category tree so subcategories can be resolved by ID
    const allCategories: Category[] = [];
    const flattenCats = (cats: Category[]) => cats.forEach(c => { allCategories.push(c); if (c.subcategories) flattenCats(c.subcategories); });
    flattenCats(categories);

    // Helper: resolve the effective (top-level) category name for a transaction.
    // Subcategories are rolled up to their parent so charts group consistently.
    const resolveCategory = (categoryId?: string | null): { name: string; topLevelName: string } => {
      if (!categoryId) return { name: 'Uncategorized', topLevelName: 'Uncategorized' };
      const cat = allCategories.find(c => c.id === categoryId);
      if (!cat) return { name: 'Uncategorized', topLevelName: 'Uncategorized' };
      if (cat.parent_id) {
        const parent = allCategories.find(c => c.id === cat.parent_id);
        return { name: cat.name, topLevelName: parent?.name || cat.name };
      }
      return { name: cat.name, topLevelName: cat.name };
    };

    // Sort archives by date and limit to the most recent 12
    const sortedArchives = [...archives]
      .sort((a, b) => new Date(a.archived_at).getTime() - new Date(b.archived_at).getTime())
      .slice(-12);

    for (const archive of sortedArchives) {
      const archiveLabel = archive.description || new Date(archive.archived_at).toLocaleDateString();

      try {
        // Fetch transactions for this archive
        const response = await axios.get(`${API_URL}/api/archives/${archive.id}/transactions`);
        const transactions: Transaction[] = response.data || [];

        // Initialize category map for this archive
        topCategories.set(archiveLabel, new Map());

        const personTotals = new Map<string, number>();

        // Process each transaction
        for (const transaction of transactions) {
          const assignedPeople = transaction.assigned_to && transaction.assigned_to.length > 0
            ? transaction.assigned_to
            : people.map(p => p.name); // If no assignment, assign to all people

          const numPeople = assignedPeople.length;
          const txSign = transaction.amount < 0 ? -1 : 1;
          const allocations = (transaction.splits && transaction.splits.length > 0)
            ? transaction.splits.map(split => ({
                categoryId: split.category_id,
                amount: txSign * Number(split.amount || 0),
              }))
            : [];

          for (const allocation of allocations) {
            const { name: categoryName, topLevelName } = resolveCategory(allocation.categoryId);

            // Skip Reimbursable category
            if (topLevelName === 'Reimbursable') {
              continue;
            }

            // Track for top categories (group by top-level category)
            const archiveCategoryMap = topCategories.get(archiveLabel)!;
            archiveCategoryMap.set(
              topLevelName,
              (archiveCategoryMap.get(topLevelName) || 0) + allocation.amount
            );

            const amountPerPerson = allocation.amount / numPeople;

            for (const personName of assignedPeople) {
              const person = people.find(p => p.name === personName);
              if (!person) {
                continue;
              }

              // Add to category spending data (store subcategory name so pie chart can drill down)
              categorySpending.push({
                archive: archiveLabel,
                person: person.name,
                amount: amountPerPerson,
                category: categoryName,
              });

              personTotals.set(personName, (personTotals.get(personName) || 0) + amountPerPerson);
            }
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

  // Flatten nested category tree for parent lookups
  const flatCategories = React.useMemo(() => {
    const flat: Category[] = [];
    const flatten = (cats: Category[]) => cats.forEach(c => { flat.push(c); if (c.subcategories) flatten(c.subcategories); });
    flatten(categories);
    return flat;
  }, [categories]);

  // Filter and group data for pie charts by person, with optional drill-down
  const getPieDataByPerson = (drillDownCategoryName?: string | null) => {
    if (!selectedArchive) return [];

    const filteredData = categorySpendingData.filter(d => d.archive === selectedArchive);

    // Roll up subcategories or drill in
    const aggregated: Record<string, Record<string, number>> = {};

    for (const d of filteredData) {
      const cat = flatCategories.find(c => c.name === d.category);

      let bucketName: string;
      if (drillDownCategoryName) {
        // Only include items belonging to the drilled-in top-level category
        const parentName = cat?.parent_id
          ? flatCategories.find(c => c.id === cat.parent_id)?.name
          : cat?.name;
        if (parentName !== drillDownCategoryName && d.category !== drillDownCategoryName) continue;
        bucketName = d.category;
      } else {
        // Roll up subcategories to parent
        if (cat?.parent_id) {
          bucketName = flatCategories.find(c => c.id === cat.parent_id)?.name || d.category;
        } else {
          bucketName = d.category;
        }
      }

      if (!aggregated[d.person]) aggregated[d.person] = {};
      aggregated[d.person][bucketName] = (aggregated[d.person][bucketName] || 0) + d.amount;
    }

    return Object.entries(aggregated)
      .map(([person, cats]) => ({
        person,
        data: Object.entries(cats)
          .map(([category, amount]) => ({ type: category, value: parseFloat(amount.toFixed(2)) }))
          .sort((a, b) => b.value - a.value),
      }))
      .sort((a, b) => {
        if (a.person === 'Joint') return -1;
        if (b.person === 'Joint') return 1;
        return a.person.localeCompare(b.person);
      });
  };

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

  const pieDataByPerson = getPieDataByPerson();

  return (
    <div style={{ padding: 24 }}>
      <Row gutter={[16, 16]} align="middle" style={{ marginBottom: 24 }}>
        <Col>
          <Title level={2} style={{ margin: 0 }}>
            <LineChartOutlined /> Spending Trends
          </Title>
        </Col>
      </Row>

      <Row style={{ marginBottom: 16 }}>
        <Col>
          <span style={{ color: '#8c8c8c', fontSize: 13 }}>
            <InfoCircleOutlined style={{ marginRight: 6 }} />
            Transactions that are categorized as "Reimbursable" are excluded from all charts.
          </span>
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
                  const drillDownCategoryName = drillDownState[person] || null;
                  const drillData = drillDownCategoryName
                    ? getPieDataByPerson(drillDownCategoryName).find(p => p.person === person)?.data || []
                    : data;
                  const personTotal = drillData.reduce((sum, d) => sum + d.value, 0);

                  // Build color map for drill-down
                  const getDrillColor = (itemType: string, index: number): string => {
                    if (!drillDownCategoryName) return getCategoryColor(itemType, categories);
                    const parentCat = flatCategories.find(c => c.name === drillDownCategoryName);
                    const baseColor = parentCat?.color || '#1890ff';
                    const variants = generateColorVariants(baseColor, drillData.length);
                    return variants[index] || baseColor;
                  };

                  return (
                    <Col xs={24} sm={24} md={12} lg={8} key={person}>
                      <Card
                        size="small"
                        title={
                          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <div>
                              <span>{person}</span>
                              {drillDownCategoryName && (
                                <div style={{ marginTop: 2 }}>
                                  <span
                                    onClick={() => setDrillDownState(prev => ({ ...prev, [person]: null }))}
                                    style={{ fontSize: 11, color: '#1890ff', cursor: 'pointer' }}
                                  >
                                    ← Back
                                  </span>
                                  <span style={{ fontSize: 11, color: '#999', marginLeft: 4 }}>
                                    {drillDownCategoryName} breakdown
                                  </span>
                                </div>
                              )}
                            </div>
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
                            key={`${person}-${selectedArchive}-${drillDownCategoryName || 'top'}`}
                            data={drillData.map((d, i) => ({
                              ...d,
                              color: getDrillColor(d.type, i),
                            }))}
                            angleField="value"
                            colorField="type"
                            radius={0.75}
                            innerRadius={0.3}
                            scale={{
                              color: {
                                range: drillData.map((d, i) => getDrillColor(d.type, i)),
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
                            const total = drillData.reduce((sum, d) => sum + d.value, 0);
                            return drillData.map((item, index) => {
                              const topCat = !drillDownCategoryName
                                ? categories.find(c => c.name === item.type)
                                : null;
                              const isDrillable = topCat && topCat.subcategories && topCat.subcategories.length > 0;
                              const color = getDrillColor(item.type, index);
                              return (
                                <div
                                  key={index}
                                  onClick={() => {
                                    if (isDrillable) {
                                      setDrillDownState(prev => ({ ...prev, [person]: item.type }));
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
                                  title={isDrillable ? `Click to drill into ${item.type}` : undefined}
                                >
                                  <div
                                    style={{
                                      width: '12px',
                                      height: '12px',
                                      borderRadius: '50%',
                                      backgroundColor: color,
                                      marginTop: '2px',
                                      flexShrink: 0
                                    }}
                                  />
                                  <div style={{ fontSize: '12px', lineHeight: '1.2', flex: 1 }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                                      <span style={{ fontWeight: 500, color: isDrillable ? '#1890ff' : '#333', textDecoration: isDrillable ? 'underline' : 'none' }}>
                                        {item.type} ({((item.value / total) * 100).toFixed(1)}%)
                                      </span>
                                      <span style={{ color: '#666', fontSize: '11px' }}>
                                        ${item.value.toFixed(2)}
                                      </span>
                                    </div>
                                  </div>
                                </div>
                              );
                            });
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

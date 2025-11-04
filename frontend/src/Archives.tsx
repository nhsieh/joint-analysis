import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Typography,
  Card,
  Table,
  Button,
  Space,
  Row,
  Col,
  Statistic,
  message,
  Spin,
  Modal,
} from 'antd';
import {
  InboxOutlined,
  DollarCircleOutlined,
  FileTextOutlined,
  EyeOutlined,
} from '@ant-design/icons';
import { ColumnsType } from 'antd/es/table';

interface Archive {
  id: string;
  name: string;
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
  date_uploaded: string;
  file_name: string;
  transaction_date: string;
  posted_date: string;
  card_number: string;
  category_id: string;
}

const { Title, Text } = Typography;
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

const Archives: React.FC = () => {
  const [archives, setArchives] = useState<Archive[]>([]);
  const [archivedTransactions, setArchivedTransactions] = useState<Transaction[]>([]);
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [selectedArchive, setSelectedArchive] = useState<Archive | null>(null);

  useEffect(() => {
    fetchArchives();
  }, []);

  const fetchArchives = async () => {
    try {
      setLoading(true);
      const response = await axios.get(`${API_URL}/api/archives`);
      setArchives(response.data || []);
    } catch (error) {
      console.error('Error fetching archives:', error);
      message.error('Error fetching archives');
    } finally {
      setLoading(false);
    }
  };

  const fetchArchivedTransactions = async (archiveId: string) => {
    try {
      setDetailLoading(true);
      const response = await axios.get(`${API_URL}/api/archives/${archiveId}/transactions`);
      setArchivedTransactions(response.data || []);
    } catch (error) {
      console.error('Error fetching archived transactions:', error);
      message.error('Error fetching archived transactions');
    } finally {
      setDetailLoading(false);
    }
  };

  const showArchiveDetails = async (archive: Archive) => {
    setSelectedArchive(archive);
    setModalVisible(true);
    await fetchArchivedTransactions(archive.id);
  };

  const archiveColumns: ColumnsType<Archive> = [
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      render: (text: string) => text || '-',
    },
    {
      title: 'Archived Date',
      dataIndex: 'archived_at',
      key: 'archived_at',
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Transaction Count',
      dataIndex: 'transaction_count',
      key: 'transaction_count',
      align: 'center',
    },
    {
      title: 'Total Amount',
      dataIndex: 'total_amount',
      key: 'total_amount',
      render: (amount: number) => `$${amount.toFixed(2)}`,
      align: 'right',
    },
    {
      title: 'Person Totals',
      dataIndex: 'person_totals',
      key: 'person_totals',
      render: (personTotals: PersonTotal[]) => (
        <div>
          {personTotals?.map(pt => (
            <div key={pt.name} style={{ fontSize: '12px' }}>
              {pt.name}: ${pt.total.toFixed(2)}
            </div>
          )) || '-'}
        </div>
      ),
      width: '200px',
    },
    {
      title: 'Actions',
      key: 'actions',
      render: (_, record: Archive) => (
        <Button
          icon={<EyeOutlined />}
          onClick={() => showArchiveDetails(record)}
          size="small"
        >
          View Transactions
        </Button>
      ),
    },
  ];

  const transactionColumns: ColumnsType<Transaction> = [
    {
      title: 'Date',
      dataIndex: 'transaction_date',
      key: 'transaction_date',
      render: (date: string) => date ? new Date(date).toLocaleDateString() : '-',
      sorter: (a, b) => {
        const dateA = a.transaction_date ? new Date(a.transaction_date).getTime() : 0;
        const dateB = b.transaction_date ? new Date(b.transaction_date).getTime() : 0;
        return dateB - dateA;
      },
      defaultSortOrder: 'ascend',
      width: '15%',
    },
    {
      title: 'Description',
      dataIndex: 'description',
      key: 'description',
      width: '40%',
    },
    {
      title: 'Amount',
      dataIndex: 'amount',
      key: 'amount',
      render: (amount: number) => `$${amount.toFixed(2)}`,
      align: 'right',
      width: '20%',
    },
    {
      title: 'Assigned To',
      dataIndex: 'assigned_to',
      key: 'assigned_to',
      render: (assignedTo: string[]) => assignedTo?.join(', ') || 'Unassigned',
      width: '25%',
    },
  ];

  // Calculate summary statistics
  const totalArchives = archives.length;
  const totalArchivedTransactions = archives.reduce((sum, archive) => sum + archive.transaction_count, 0);
  const totalArchivedAmount = archives.reduce((sum, archive) => sum + archive.total_amount, 0);

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>
        <InboxOutlined style={{ marginRight: 8 }} />
        Archives
      </Title>

      {/* Summary Statistics */}
      <Row gutter={16} style={{ marginBottom: '24px' }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="Total Archives"
              value={totalArchives}
              prefix={<InboxOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Archived Transactions"
              value={totalArchivedTransactions}
              prefix={<FileTextOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="Total Archived Amount"
              value={totalArchivedAmount}
              prefix={<DollarCircleOutlined />}
              precision={2}
            />
          </Card>
        </Col>
      </Row>

      {/* Archives Table */}
      <Card
        title={`Archives (${totalArchives})`}
        variant="borderless"
        hoverable
      >
        <Spin spinning={loading}>
          <Table
            columns={archiveColumns}
            dataSource={archives}
            rowKey="id"
            pagination={{
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total, range) =>
                `${range[0]}-${range[1]} of ${total} archives`,
            }}
          />
        </Spin>
      </Card>

      {/* Archive Details Modal */}
      <Modal
        title={selectedArchive ? `${selectedArchive.name} - Transactions` : 'Archive Details'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={1000}
      >
        {selectedArchive && (
          <div>
            <Row gutter={16} style={{ marginBottom: '16px' }}>
              <Col span={6}>
                <Statistic
                  title="Archive Date"
                  value={new Date(selectedArchive.archived_at).toLocaleDateString()}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="Transaction Count"
                  value={selectedArchive.transaction_count}
                />
              </Col>
              <Col span={6}>
                <Statistic
                  title="Total Amount"
                  value={selectedArchive.total_amount}
                  precision={2}
                  prefix="$"
                />
              </Col>
              <Col span={6}>
                <Card size="small" title="Person Totals">
                  {selectedArchive.person_totals?.map(pt => (
                    <div key={pt.name} style={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Text>{pt.name}:</Text>
                      <Text strong>${pt.total.toFixed(2)}</Text>
                    </div>
                  )) || <Text type="secondary">No individual totals</Text>}
                </Card>
              </Col>
            </Row>

            <Spin spinning={detailLoading}>
              <Table
                columns={transactionColumns}
                dataSource={archivedTransactions}
                rowKey="id"
                pagination={{
                  pageSize: 10,
                  showSizeChanger: false,
                }}
                size="small"
              />
            </Spin>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default Archives;
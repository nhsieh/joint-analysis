import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Typography,
  Card,
  Button,
  Space,
  Row,
  Col,
  message,
  Modal,
  Form,
  Input,
  ColorPicker,
  Popconfirm,
} from 'antd';
import {
  EditOutlined,
  PlusOutlined,
  TagOutlined,
  DeleteOutlined,
  UserAddOutlined,
} from '@ant-design/icons';

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

const { Title, Text } = Typography;
const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8081';

const Settings: React.FC = () => {
  const [people, setPeople] = useState<Person[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [newPersonName, setNewPersonName] = useState('');
  const [categoryModalVisible, setCategoryModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [categoryForm] = Form.useForm();

  useEffect(() => {
    fetchPeople();
    fetchCategories();
  }, []);

  const fetchPeople = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/people`);
      setPeople(response.data || []);
    } catch (error) {
      console.error('Error fetching people:', error);
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
    Modal.confirm({
      title: 'Delete Person',
      content: `Are you sure you want to delete "${personName}"? This action cannot be undone and will affect all transactions assigned to this person (including archived transactions).`,
      okText: 'Yes, Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await axios.delete(`${API_URL}/api/people/${personId}`);
          message.success(`${personName} deleted successfully!`);
          fetchPeople();
        } catch (error) {
          console.error('Error deleting person:', error);
          message.error('Error deleting person');
        }
      },
    });
  };

  const fetchCategories = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/categories`);
      setCategories(response.data || []);
    } catch (error) {
      console.error('Error fetching categories:', error);
    }
  };

  // Category management functions
  const openCategoryModal = (category?: Category) => {
    setEditingCategory(category || null);
    setCategoryModalVisible(true);
    if (category) {
      categoryForm.setFieldsValue({
        name: category.name,
        description: category.description || '',
        color: category.color || '#1890ff',
      });
    } else {
      categoryForm.resetFields();
    }
  };

  const closeCategoryModal = () => {
    setCategoryModalVisible(false);
    setEditingCategory(null);
    categoryForm.resetFields();
  };

  const handleCategorySubmit = async (values: any) => {
    try {
      const categoryData = {
        name: values.name,
        description: values.description || '',
        color: values.color?.toHexString?.() || values.color || '#1890ff',
      };

      if (editingCategory) {
        // Update existing category
        await axios.put(`${API_URL}/api/categories/${editingCategory.id}`, categoryData);
        message.success('Category updated successfully!');
      } else {
        // Create new category
        await axios.post(`${API_URL}/api/categories`, categoryData);
        message.success('Category created successfully!');
      }

      fetchCategories();
      closeCategoryModal();
    } catch (error) {
      console.error('Error saving category:', error);
      message.error(`Error ${editingCategory ? 'updating' : 'creating'} category`);
    }
  };

  const deleteCategory = async (categoryId: string, categoryName: string) => {
    try {
      await axios.delete(`${API_URL}/api/categories/${categoryId}`);
      message.success(`${categoryName} deleted successfully!`);
      fetchCategories();
    } catch (error) {
      console.error('Error deleting category:', error);
      message.error('Error deleting category');
    }
  };

  return (
    <div style={{ padding: '24px' }}>
      <Title level={2}>Settings</Title>

      {/* People Management Section */}
      <Card
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>
              <UserAddOutlined style={{ marginRight: 8 }} />
              People ({people.length})
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
      >
        {people.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <Text type="secondary">
              No people added yet. Add people to start tracking expenses.
            </Text>
          </div>
        ) : (
          <Row gutter={[8, 8]}>
            {people.map((person) => (
              <Col xs={12} sm={8} md={6} lg={4} xl={3} key={person.id}>
                <Card
                  size="small"
                  style={{
                    textAlign: 'center',
                    minHeight: '60px',
                  }}
                  bodyStyle={{ padding: '8px' }}
                  hoverable
                >
                  <div style={{
                    fontWeight: 'bold',
                    fontSize: '14px',
                    marginBottom: '4px',
                    whiteSpace: 'nowrap',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis'
                  }}>
                    {person.name}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'center', gap: '4px' }}>
                    <Popconfirm
                      title="Delete Person"
                      description={`Are you sure you want to delete "${person.name}"?`}
                      onConfirm={() => deletePerson(person.id, person.name)}
                      okText="Yes"
                      cancelText="No"
                    >
                      <Button
                        type="text"
                        danger
                        icon={<DeleteOutlined />}
                        size="small"
                        style={{ padding: '2px 4px', minWidth: 'auto' }}
                      />
                    </Popconfirm>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>
        )}
      </Card>

      {/* Categories Management Section */}
      <Card
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>
              <TagOutlined style={{ marginRight: 8 }} />
              Categories ({categories.length})
            </span>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              size="middle"
              onClick={() => openCategoryModal()}
            >
              Add Category
            </Button>
          </div>
        }
        style={{ marginBottom: 24 }}
      >
        <Row gutter={[8, 8]}>
          {categories.map((category) => (
            <Col xs={12} sm={8} md={6} lg={4} xl={3} key={category.id}>
              <Card
                size="small"
                style={{
                  textAlign: 'center',
                  borderColor: category.color,
                  borderWidth: 1,
                  minHeight: '80px',
                }}
                bodyStyle={{ padding: '8px' }}
                hoverable
              >
                <div style={{
                  background: category.color,
                  color: 'white',
                  padding: '4px 8px',
                  borderRadius: '4px',
                  marginBottom: '4px',
                  fontWeight: 'bold',
                  fontSize: '12px',
                  whiteSpace: 'nowrap',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis'
                }}>
                  {category.name}
                </div>
                {category.description && (
                  <Text
                    type="secondary"
                    style={{
                      fontSize: '10px',
                      display: 'block',
                      lineHeight: '1.2',
                      marginBottom: '4px',
                      height: '24px',
                      overflow: 'hidden',
                      textOverflow: 'ellipsis'
                    }}
                  >
                    {category.description}
                  </Text>
                )}
                <div style={{ display: 'flex', justifyContent: 'center', gap: '4px' }}>
                  <Button
                    type="text"
                    icon={<EditOutlined />}
                    size="small"
                    style={{ padding: '2px 4px', minWidth: 'auto' }}
                    onClick={() => openCategoryModal(category)}
                  />
                  <Popconfirm
                    title="Delete Category"
                    description={`Are you sure you want to delete "${category.name}"?`}
                    onConfirm={() => deleteCategory(category.id, category.name)}
                    okText="Yes"
                    cancelText="No"
                  >
                    <Button
                      type="text"
                      danger
                      icon={<DeleteOutlined />}
                      size="small"
                      style={{ padding: '2px 4px', minWidth: 'auto' }}
                    />
                  </Popconfirm>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>

      {/* Category Modal */}
      <Modal
        title={editingCategory ? 'Edit Category' : 'Add Category'}
        open={categoryModalVisible}
        onCancel={closeCategoryModal}
        footer={null}
        width={500}
      >
        <Form
          form={categoryForm}
          layout="vertical"
          onFinish={handleCategorySubmit}
        >
          <Form.Item
            name="name"
            label="Category Name"
            rules={[{ required: true, message: 'Please enter a category name' }]}
          >
            <Input placeholder="Enter category name" />
          </Form.Item>

          <Form.Item
            name="description"
            label="Description (Optional)"
          >
            <Input.TextArea
              placeholder="Enter category description"
              rows={3}
            />
          </Form.Item>

          <Form.Item
            name="color"
            label="Color"
            rules={[{ required: true, message: 'Please select a color' }]}
          >
            <ColorPicker
              showText
              presets={[
                {
                  label: 'Recommended',
                  colors: [
                    '#FF7043', '#42A5F5', '#AB47BC', '#66BB6A', '#FFA726',
                    '#EF5350', '#26C6DA', '#8D6E63', '#78909C', '#1890ff',
                  ],
                },
              ]}
            />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={closeCategoryModal}>
                Cancel
              </Button>
              <Button type="primary" htmlType="submit">
                {editingCategory ? 'Update' : 'Create'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Settings;
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
  Select,
} from 'antd';
import {
  EditOutlined,
  PlusOutlined,
  TagOutlined,
  DeleteOutlined,
  UserAddOutlined,
  OrderedListOutlined,
} from '@ant-design/icons';
import { Category, Rule } from './types';

interface Person {
  id: string;
  name: string;
  email?: string;
  created_at: string;
  updated_at: string;
}

const { Title, Text } = Typography;
const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8081';

const Settings: React.FC = () => {
  const [people, setPeople] = useState<Person[]>([]);
  const [categories, setCategories] = useState<Category[]>([]);
  const [rules, setRules] = useState<Rule[]>([]);
  const [ruleModalVisible, setRuleModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<Rule | null>(null);
  const [ruleForm] = Form.useForm();
  const [newPersonName, setNewPersonName] = useState('');
  const [categoryModalVisible, setCategoryModalVisible] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  // parentId is set when adding a subcategory to a parent
  const [subcategoryParentId, setSubcategoryParentId] = useState<string | null>(null);
  const [categoryForm] = Form.useForm();

  useEffect(() => {
    fetchPeople();
    fetchCategories();
    fetchRules();
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

  const fetchRules = async () => {
    try {
      const response = await axios.get(`${API_URL}/api/rules`);
      setRules(response.data || []);
    } catch (error) {
      console.error('Error fetching rules:', error);
    }
  };

  const openRuleModal = (rule?: Rule) => {
    setEditingRule(rule || null);
    setRuleModalVisible(true);
    if (rule) {
      ruleForm.setFieldsValue({
        match_value: rule.match_value,
        category_id: rule.category_id,
        priority: rule.priority,
      });
    } else {
      ruleForm.resetFields();
    }
  };

  const closeRuleModal = () => {
    setRuleModalVisible(false);
    setEditingRule(null);
    ruleForm.resetFields();
  };

  const handleRuleSubmit = async (values: any) => {
    try {
      if (editingRule) {
        await axios.put(`${API_URL}/api/rules/${editingRule.id}`, {
          match_value: values.match_value,
          category_id: values.category_id,
          priority: Number(values.priority),
        });
        message.success('Rule updated successfully!');
      } else {
        await axios.post(`${API_URL}/api/rules`, {
          match_value: values.match_value,
          category_id: values.category_id,
          priority: Number(values.priority),
        });
        message.success('Rule created successfully!');
      }
      fetchRules();
      closeRuleModal();
    } catch (error) {
      console.error('Error saving rule:', error);
      message.error(`Error ${editingRule ? 'updating' : 'creating'} rule`);
    }
  };

  const handleDeleteRule = (ruleId: string, matchValue: string) => {
    Modal.confirm({
      title: 'Delete Rule',
      content: `Are you sure you want to delete the rule for "${matchValue}"?`,
      okText: 'Yes, Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await axios.delete(`${API_URL}/api/rules/${ruleId}`);
          message.success('Rule deleted successfully!');
          fetchRules();
        } catch (error) {
          console.error('Error deleting rule:', error);
          message.error('Error deleting rule');
        }
      },
    });
  };

  // Flatten categories for select options (include subcategories)
  const flatCategories = categories.reduce<Category[]>((acc, cat) => {
    acc.push(cat);
    if (cat.subcategories) acc.push(...cat.subcategories);
    return acc;
  }, []);

  // Open modal for top-level category create/edit
  const openCategoryModal = (category?: Category) => {
    setEditingCategory(category || null);
    setSubcategoryParentId(null);
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

  // Open modal pre-filled to add a subcategory under a given parent
  const openSubcategoryModal = (parent: Category, subToEdit?: Category) => {
    setSubcategoryParentId(parent.id);
    setEditingCategory(subToEdit || null);
    setCategoryModalVisible(true);
    if (subToEdit) {
      categoryForm.setFieldsValue({
        name: subToEdit.name,
        description: subToEdit.description || '',
      });
    } else {
      categoryForm.resetFields();
    }
  };

  const closeCategoryModal = () => {
    setCategoryModalVisible(false);
    setEditingCategory(null);
    setSubcategoryParentId(null);
    categoryForm.resetFields();
  };

  const handleCategorySubmit = async (values: any) => {
    try {
      const isSubcategory = subcategoryParentId !== null;

      const categoryData: Record<string, any> = {
        name: values.name,
        description: values.description || '',
      };

      if (!isSubcategory) {
        // Top-level: apply color from picker
        categoryData.color = values.color?.toHexString?.() || values.color || '#1890ff';
      }

      if (editingCategory) {
        await axios.put(`${API_URL}/api/categories/${editingCategory.id}`, categoryData);
        message.success('Category updated successfully!');
      } else {
        if (isSubcategory) {
          categoryData.parent_id = subcategoryParentId;
        }
        await axios.post(`${API_URL}/api/categories`, categoryData);
        message.success(isSubcategory ? 'Subcategory created successfully!' : 'Category created successfully!');
      }

      fetchCategories();
      closeCategoryModal();
    } catch (error) {
      console.error('Error saving category:', error);
      message.error(`Error ${editingCategory ? 'updating' : 'creating'} category`);
    }
  };

  const deleteCategory = async (categoryId: string, categoryName: string, hasSubcategories?: boolean) => {
    const extra = hasSubcategories
      ? ' Its subcategories will also be deleted.'
      : '';
    Modal.confirm({
      title: 'Delete Category',
      content: `Are you sure you want to delete "${categoryName}"?${extra} This action cannot be undone.`,
      okText: 'Yes, Delete',
      okType: 'danger',
      cancelText: 'Cancel',
      onOk: async () => {
        try {
          await axios.delete(`${API_URL}/api/categories/${categoryId}`);
          message.success(`${categoryName} deleted successfully!`);
          fetchCategories();
        } catch (error) {
          console.error('Error deleting category:', error);
          message.error('Error deleting category');
        }
      },
    });
  };

  const isSubcategoryModal = subcategoryParentId !== null;
  // Count total top-level categories
  const totalCategoryCount = categories.length;

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
              Categories ({totalCategoryCount})
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
          {categories.map((category) => {
            const subs = category.subcategories || [];
            return (
              <Col xs={24} sm={12} md={8} lg={6} key={category.id}>
                {/* Top-level category card */}
                <Card
                  size="small"
                  style={{
                    borderColor: category.color,
                    borderWidth: 1,
                    marginBottom: 4,
                  }}
                  bodyStyle={{ padding: '8px' }}
                  hoverable
                >
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <div style={{
                      background: category.color,
                      color: 'white',
                      padding: '2px 8px',
                      borderRadius: '4px',
                      fontWeight: 'bold',
                      fontSize: '12px',
                      flex: 1,
                      marginRight: 4,
                      overflow: 'hidden',
                      textOverflow: 'ellipsis',
                      whiteSpace: 'nowrap',
                    }}>
                      {category.name}
                    </div>
                    <Space size={2}>
                      <Button
                        type="text"
                        icon={<EditOutlined />}
                        size="small"
                        style={{ padding: '2px 4px', minWidth: 'auto' }}
                        onClick={() => openCategoryModal(category)}
                      />
                      <Button
                        type="text"
                        icon={<PlusOutlined />}
                        size="small"
                        title="Add Subcategory"
                        style={{ padding: '2px 4px', minWidth: 'auto' }}
                        onClick={() => openSubcategoryModal(category)}
                      />
                      <Popconfirm
                        title="Delete Category"
                        description={
                          subs.length > 0
                            ? `Are you sure? "${category.name}" and its ${subs.length} subcategori${subs.length === 1 ? 'y' : 'es'} will be deleted.`
                            : `Are you sure you want to delete "${category.name}"?`
                        }
                        onConfirm={() => deleteCategory(category.id, category.name, subs.length > 0)}
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
                    </Space>
                  </div>
                  {category.description && (
                    <Text
                      type="secondary"
                      style={{ fontSize: '10px', display: 'block', marginTop: 2, lineHeight: '1.2' }}
                    >
                      {category.description}
                    </Text>
                  )}

                  {/* Subcategory list */}
                  {subs.length > 0 && (
                    <div style={{ marginTop: 6, paddingLeft: 8, borderLeft: `2px solid ${category.color}` }}>
                      {subs.map((sub) => (
                        <div
                          key={sub.id}
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'space-between',
                            padding: '2px 0',
                          }}
                        >
                          <Text style={{ fontSize: '11px', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                            {sub.name}
                          </Text>
                          <Space size={2}>
                            <Button
                              type="text"
                              icon={<EditOutlined />}
                              size="small"
                              style={{ padding: '1px 3px', minWidth: 'auto', fontSize: '10px' }}
                              onClick={() => openSubcategoryModal(category, sub)}
                            />
                            <Popconfirm
                              title="Delete Subcategory"
                              description={`Are you sure you want to delete "${sub.name}"?`}
                              onConfirm={() => deleteCategory(sub.id, sub.name)}
                              okText="Yes"
                              cancelText="No"
                            >
                              <Button
                                type="text"
                                danger
                                icon={<DeleteOutlined />}
                                size="small"
                                style={{ padding: '1px 3px', minWidth: 'auto', fontSize: '10px' }}
                              />
                            </Popconfirm>
                          </Space>
                        </div>
                      ))}
                    </div>
                  )}
                </Card>
              </Col>
            );
          })}
        </Row>
      </Card>

      {/* Rules Management Section */}
      <Card
        title={
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span>
              <OrderedListOutlined style={{ marginRight: 8 }} />
              Rules ({rules.length})
            </span>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              size="middle"
              onClick={() => openRuleModal()}
            >
              Add Rule
            </Button>
          </div>
        }
        style={{ marginBottom: 24 }}
      >
        {rules.length === 0 ? (
          <div style={{ textAlign: 'center', padding: '40px 0' }}>
            <Text type="secondary">
              No categorization rules yet. Add rules to auto-categorize imported transactions.
            </Text>
          </div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid #f0f0f0' }}>
                <th style={{ textAlign: 'left', padding: '8px', fontWeight: 600 }}>Match Value</th>
                <th style={{ textAlign: 'left', padding: '8px', fontWeight: 600 }}>Category</th>
                <th style={{ textAlign: 'center', padding: '8px', fontWeight: 600 }}>Priority</th>
                <th style={{ textAlign: 'right', padding: '8px', fontWeight: 600 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id} style={{ borderBottom: '1px solid #f0f0f0' }}>
                  <td style={{ padding: '8px', fontFamily: 'monospace' }}>{rule.match_value}</td>
                  <td style={{ padding: '8px' }}>{rule.category_name}</td>
                  <td style={{ padding: '8px', textAlign: 'center' }}>{rule.priority}</td>
                  <td style={{ padding: '8px', textAlign: 'right' }}>
                    <Space size={4}>
                      <Button
                        type="text"
                        icon={<EditOutlined />}
                        size="small"
                        onClick={() => openRuleModal(rule)}
                      />
                      <Popconfirm
                        title="Delete Rule"
                        description={`Delete rule for "${rule.match_value}"?`}
                        onConfirm={() => handleDeleteRule(rule.id, rule.match_value)}
                        okText="Yes"
                        cancelText="No"
                      >
                        <Button
                          type="text"
                          danger
                          icon={<DeleteOutlined />}
                          size="small"
                        />
                      </Popconfirm>
                    </Space>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </Card>

      {/* Rule Modal */}
      <Modal
        title={editingRule ? 'Edit Rule' : 'Add Rule'}
        open={ruleModalVisible}
        onCancel={closeRuleModal}
        footer={null}
        width={480}
      >
        <Form
          form={ruleForm}
          layout="vertical"
          onFinish={handleRuleSubmit}
          initialValues={{ priority: 0 }}
        >
          <Form.Item
            name="match_value"
            label="Match Value"
            rules={[{ required: true, message: 'Please enter a match value' }]}
          >
            <Input placeholder="e.g. Trader Joe, Whole Foods, Dining" />
          </Form.Item>

          <Form.Item
            name="category_id"
            label="Category"
            rules={[{ required: true, message: 'Please select a category' }]}
          >
            <Select placeholder="Select a category" showSearch optionFilterProp="label">
              {flatCategories.map((cat) => (
                <Select.Option key={cat.id} value={cat.id} label={cat.name}>
                  {cat.parent_id ? `  ↳ ${cat.name}` : cat.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="priority"
            label="Priority (lower = higher priority)"
            rules={[{ required: true, message: 'Please enter a priority' }]}
          >
            <Input type="number" placeholder="0" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={closeRuleModal}>Cancel</Button>
              <Button type="primary" htmlType="submit">
                {editingRule ? 'Update' : 'Create'}
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* Category / Subcategory Modal */}
      <Modal
        title={
          editingCategory
            ? isSubcategoryModal ? 'Edit Subcategory' : 'Edit Category'
            : isSubcategoryModal ? 'Add Subcategory' : 'Add Category'
        }
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
            label={isSubcategoryModal ? 'Subcategory Name' : 'Category Name'}
            rules={[{ required: true, message: `Please enter a ${isSubcategoryModal ? 'subcategory' : 'category'} name` }]}
          >
            <Input placeholder={`Enter ${isSubcategoryModal ? 'subcategory' : 'category'} name`} />
          </Form.Item>

          <Form.Item
            name="description"
            label="Description (Optional)"
          >
            <Input.TextArea
              placeholder="Enter description"
              rows={3}
            />
          </Form.Item>

          {/* Color picker only for top-level categories */}
          {!isSubcategoryModal && (
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
          )}

          {isSubcategoryModal && (
            <Text type="secondary" style={{ fontSize: '12px', display: 'block', marginBottom: 16 }}>
              Color is inherited from the parent category.
            </Text>
          )}

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
